package app

import (
	"context"
	"flag"
	"time"

	cacheddiscovery "k8s.io/client-go/discovery/cached"
	kanariniclientset "github.com/nilebox/kanarini/pkg/client/clientset_generated/clientset"
	kanariniclientset_typed "github.com/nilebox/kanarini/pkg/client/clientset_generated/clientset/typed/kanarini/v1alpha1"
	kanariniinformers "github.com/nilebox/kanarini/pkg/client/informers_generated/externalversions"
	"github.com/nilebox/kanarini/pkg/controller"
	"go.uber.org/zap"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/nilebox/kanarini/pkg/metrics"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"k8s.io/metrics/pkg/client/external_metrics"
	"k8s.io/client-go/restmapper"
	"k8s.io/apimachinery/pkg/util/wait"
)

type App struct {
	Logger *zap.Logger

	MainClient kubernetes.Interface

	GenericControllerOptions
	LeaderElectionOptions
	RestClientOptions
	LoggerOptions

	RestConfig   *rest.Config
	ResyncPeriod time.Duration
}

func NewFromFlags(name string, flagset *flag.FlagSet, arguments []string) (*App, error) {
	a := App{}

	BindGenericControllerFlags(&a.GenericControllerOptions, flagset)
	BindLeaderElectionFlags(name, &a.LeaderElectionOptions, flagset)
	BindRestClientFlags(&a.RestClientOptions, flagset)
	BindLoggerFlags(&a.LoggerOptions, flagset)

	err := flagset.Parse(arguments)
	if err != nil {
		return nil, err
	}

	a.RestConfig, err = LoadRestClientConfig(name, a.RestClientOptions)
	if err != nil {
		return nil, err
	}

	a.Logger = LoggerFromOptions(a.LoggerOptions)

	// Clients
	a.MainClient, err = kubernetes.NewForConfig(a.RestConfig)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.Logger.Sync() // nolint: errcheck
	// unhandled error above, but we are terminating anyway

	// Build the informer factory for core resources
	coreInformerFactory := informers.NewSharedInformerFactory(
		a.MainClient,
		a.ResyncPeriod,
	)
	appsSharedInformers := coreInformerFactory.Apps().V1()
	coreSharedInformers := coreInformerFactory.Core().V1()

	kanariniClientset, err := kanariniclientset.NewForConfig(a.RestConfig)
	if err != nil {
		return err
	}
	kanariniClient := kanariniclientset_typed.New(kanariniClientset.RESTClient())

	// Use a discovery client capable of being refreshed.
	cachedClient := cacheddiscovery.NewMemCacheClient(a.MainClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)
	go wait.Until(func() {
		restMapper.Reset()
	}, 30*time.Second, ctx.Done())
	apiVersionsGetter := custom_metrics.NewAvailableAPIsGetter(a.MainClient.Discovery())
	// invalidate the discovery information roughly once per resync interval our API
	// information is *at most* two resync intervals old.
	go custom_metrics.PeriodicallyInvalidate(
		apiVersionsGetter,
		45 * time.Second, // TODO: make configurable
		ctx.Done())

	metricsClient := metrics.NewRESTMetricsClient(
		custom_metrics.NewForConfig(a.RestConfig, restMapper, apiVersionsGetter),
		external_metrics.NewForConfigOrDie(a.RestConfig),
	)

	// Build the informer factory for kanarini resources
	kanariniInformerFactory := kanariniinformers.NewSharedInformerFactory(
		kanariniClientset,
		a.ResyncPeriod,
	)
	kanariniSharedInformers := kanariniInformerFactory.Kanarini().V1alpha1()

	// Informers
	canaryDeploymentInf := kanariniSharedInformers.CanaryDeployments()
	deploymentInf := appsSharedInformers.Deployments()
	serviceInf := coreSharedInformers.Services()

	c, err := controller.NewController(a.MainClient, kanariniClient, metricsClient, canaryDeploymentInf, deploymentInf, serviceInf)

	kanariniInformerFactory.Start(ctx.Done())
	coreInformerFactory.Start(ctx.Done())
	c.Run(a.Workers, ctx.Done())
	return nil
}
