package app

import (
	"context"
	"os"
	"time"
	"go.uber.org/zap"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_v1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"flag"
	"github.com/nilebox/kanarini/pkg/util/logz"
)

const (
	defaultLeaseDuration = 15 * time.Second
	defaultRenewDeadline = 10 * time.Second
	defaultRetryPeriod   = 2 * time.Second
)

// See k8s.io/apiserver/pkg/apis/config/types.go LeaderElectionConfiguration
// for leader election configuration description.
type LeaderElectionOptions struct {
	LeaderElect        bool
	LeaseDuration      time.Duration
	RenewDeadline      time.Duration
	RetryPeriod        time.Duration
	ConfigMapNamespace string
	ConfigMapName      string
}

// DoLeaderElection starts leader election and blocks until it acquires the lease.
// Returned context is cancelled once the lease is lost or ctx signals done.
func DoLeaderElection(ctx context.Context, logger *zap.Logger, component string, config LeaderElectionOptions, configMapsGetter core_v1client.ConfigMapsGetter, recorder record.EventRecorder) (context.Context, error) {
	id, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	ctxRet, cancel := context.WithCancel(ctx)
	startedLeading := make(chan struct{})
	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.ConfigMapLock{
			ConfigMapMeta: meta_v1.ObjectMeta{
				Namespace: config.ConfigMapNamespace,
				Name:      config.ConfigMapName,
			},
			Client: configMapsGetter,
			LockConfig: resourcelock.ResourceLockConfig{
				Identity:      id + "-" + component,
				EventRecorder: recorder,
			},
		},
		LeaseDuration: config.LeaseDuration,
		RenewDeadline: config.RenewDeadline,
		RetryPeriod:   config.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Info("Started leading")
				close(startedLeading)
			},
			OnStoppedLeading: func() {
				logger.Info("Leader status lost")
				cancel()
			},
		},
	})
	if err != nil {
		cancel()
		return nil, err
	}
	go func() {
		// note: because le.Run() also adds a logging panic handler panics with be logged 3 times
		defer logz.LogStructuredPanic()
		le.Run(ctx)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-startedLeading:
		return ctxRet, nil
	}
}

func BindLeaderElectionFlags(component string, o *LeaderElectionOptions, fs *flag.FlagSet) {
	// This flag is off by default only because leader election package says it is ALPHA API.
	fs.BoolVar(&o.LeaderElect, "leader-elect", false, ""+
		"Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated "+
		"components for high availability")
	fs.DurationVar(&o.LeaseDuration, "leader-elect-lease-duration", defaultLeaseDuration, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled")
	fs.DurationVar(&o.RenewDeadline, "leader-elect-renew-deadline", defaultRenewDeadline, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled")
	fs.DurationVar(&o.RetryPeriod, "leader-elect-retry-period", defaultRetryPeriod, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled")
	fs.StringVar(&o.ConfigMapNamespace, "leader-elect-configmap-namespace", meta_v1.NamespaceDefault,
		"Namespace to use for leader election ConfigMap. This is only applicable if leader election is enabled")
	fs.StringVar(&o.ConfigMapName, "leader-elect-configmap-name", component+"-leader-elect",
		"ConfigMap name to use for leader election. This is only applicable if leader election is enabled")
}
