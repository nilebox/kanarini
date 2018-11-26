package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	kanarini "github.com/nilebox/kanarini/pkg/apis/kanarini/v1alpha1"
	kanariniclientset "github.com/nilebox/kanarini/pkg/client/clientset_generated/clientset/typed/kanarini/v1alpha1"
	informers "github.com/nilebox/kanarini/pkg/client/informers_generated/externalversions/kanarini/v1alpha1"
	listers "github.com/nilebox/kanarini/pkg/client/listers_generated/kanarini/v1alpha1"
	"github.com/nilebox/kanarini/pkg/kubernetes/pkg/controller"
	"github.com/nilebox/kanarini/pkg/kubernetes/pkg/util/metrics"
	metricsclient "github.com/nilebox/kanarini/pkg/metrics"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	// maxRetries is the number of times a resource add/update will be retried before it is dropped out of the queue.
	// With the current rate-limiter in use (5ms*2^(maxRetries-1)) the following numbers represent the times
	// a resource is going to be requeued:
	//
	// 5ms, 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.3s, 2.6s, 5.1s, 10.2s, 20.4s, 41s, 82s
	maxRetries = 15
)

// controllerKind contains the schema.GroupVersionKind for this canaryDeploymentController type.
var canaryDeploymentKind = kanarini.CanaryDeploymentGVK

// CanaryDeploymentController is responsible for synchronizing CanaryDeployment objects stored
// in the system with actual running deployments and services.
type CanaryDeploymentController struct {
	kubeClient     kubernetes.Interface
	kanariniClient kanariniclientset.KanariniV1alpha1Interface
	metricsClient  metricsclient.MetricsClient

	// To allow injection of syncDeployment for testing.
	syncHandler func(dKey string) error
	// used for unit testing
	enqueueCanaryDeployment func(deployment *kanarini.CanaryDeployment)

	// cdLister can list/get canary deployments from the shared informer's store
	cdLister listers.CanaryDeploymentLister
	// dLister can list/get deployments from the shared informer's store
	dLister appslisters.DeploymentLister

	// cdListerSynced returns true if the CanaryDeployment store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	cdListerSynced cache.InformerSynced
	// dListerSynced returns true if the Deployment store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	dListerSynced cache.InformerSynced

	// CanaryDeployments that need to be synced
	queue         workqueue.RateLimitingInterface
	eventRecorder record.EventRecorder
}

// NewController returns a new CanaryDeployment canaryDeploymentController.
func NewController(
	kubeClient kubernetes.Interface,
	kanariniClient kanariniclientset.KanariniV1alpha1Interface,
	metricsClient metricsclient.MetricsClient,
	cdInformer informers.CanaryDeploymentInformer,
	dInformer appsinformers.DeploymentInformer,
) (*CanaryDeploymentController, error) {
	// Create Event broadcaster
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	// Create Events Scheme
	eventsScheme := runtime.NewScheme()
	var err error
	if err = corev1.AddToScheme(eventsScheme); err != nil {
		return nil, err
	}
	if err = kanarini.AddToScheme(eventsScheme); err != nil {
		return nil, err
	}

	if kubeClient != nil && kubeClient.CoreV1().RESTClient().GetRateLimiter() != nil {
		if err := metrics.RegisterMetricAndTrackRateLimiterUsage("canary_deployment_controller", kubeClient.CoreV1().RESTClient().GetRateLimiter()); err != nil {
			return nil, err
		}
	}
	if kanariniClient != nil && kanariniClient.RESTClient().GetRateLimiter() != nil {
		if err := metrics.RegisterMetricAndTrackRateLimiterUsage("canary_deployment_controller", kanariniClient.RESTClient().GetRateLimiter()); err != nil {
			return nil, err
		}
	}
	cdc := &CanaryDeploymentController{
		kubeClient:     kubeClient,
		kanariniClient: kanariniClient,
		metricsClient:  metricsClient,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "canary-deployment"),
		eventRecorder:  eventBroadcaster.NewRecorder(eventsScheme, v1.EventSource{Component: "canary-deployment-cdc"}),
	}

	cdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cdc.addCanaryDeployment,
		UpdateFunc: cdc.updateCanaryDeployment,
		// This will enter the sync loop and no-op, because the deployment has been deleted from the store.
		DeleteFunc: cdc.deleteCanaryDeployment,
	})
	// Subscribe to Deployment events to trigger re-processing of parent CanaryDeployment
	dInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    cdc.addDeployment,
		UpdateFunc: cdc.updateDeployment,
		// This will enter the sync loop and no-op, because the deployment has been deleted from the store.
		DeleteFunc: cdc.deleteDeployment,
	})
	cdc.syncHandler = cdc.syncDeployment
	cdc.enqueueCanaryDeployment = cdc.enqueue

	cdc.cdLister = cdInformer.Lister()
	cdc.dLister = dInformer.Lister()
	cdc.cdListerSynced = cdInformer.Informer().HasSynced
	cdc.dListerSynced = dInformer.Informer().HasSynced
	return cdc, nil
}

// Run begins watching and syncing.
func (c *CanaryDeploymentController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	glog.Info("Starting deployment controller")
	defer glog.Info("Shutting down deployment controller")

	if !controller.WaitForCacheSync("canary-deployment", stopCh, c.cdListerSynced, c.dListerSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *CanaryDeploymentController) addCanaryDeployment(obj interface{}) {
	d := obj.(*kanarini.CanaryDeployment)
	glog.V(4).Infof("Adding canary deployment %s", d.Name)
	c.enqueueCanaryDeployment(d)
}

func (c *CanaryDeploymentController) updateCanaryDeployment(old, cur interface{}) {
	oldD := old.(*kanarini.CanaryDeployment)
	curD := cur.(*kanarini.CanaryDeployment)
	glog.V(4).Infof("Updating canary deployment %s", oldD.Name)
	c.enqueueCanaryDeployment(curD)
}

func (c *CanaryDeploymentController) deleteCanaryDeployment(obj interface{}) {
	d, ok := obj.(*kanarini.CanaryDeployment)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*kanarini.CanaryDeployment)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Deployment %#v", obj))
			return
		}
	}
	glog.V(4).Infof("Deleting canary deployment %s", d.Name)
	c.enqueueCanaryDeployment(d)
}

func (c *CanaryDeploymentController) addDeployment(obj interface{}) {
	d := obj.(*apps.Deployment)
	glog.V(4).Infof("Adding deployment %s", d.Name)

	c.processDeploymentEvent(d)
}

func (c *CanaryDeploymentController) updateDeployment(old, cur interface{}) {
	oldD := old.(*apps.Deployment)
	curD := cur.(*apps.Deployment)
	glog.V(4).Infof("Updating deployment %s", oldD.Name)
	c.processDeploymentEvent(curD)
}

func (c *CanaryDeploymentController) deleteDeployment(obj interface{}) {
	d, ok := obj.(*apps.Deployment)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*apps.Deployment)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Deployment %#v", obj))
			return
		}
	}
	glog.V(4).Infof("Deleting deployment %s", d.Name)
	c.processDeploymentEvent(d)
}

func (c *CanaryDeploymentController) processDeploymentEvent(d *apps.Deployment) {
	ownerRef := metav1.GetControllerOf(d)
	if ownerRef == nil {
		return
	}
	if ownerRef.Kind != kanarini.CanaryDeploymentResourceKind {
		return
	}
	cd, err := c.cdLister.CanaryDeployments(d.Namespace).Get(ownerRef.Name)
	if err != nil {
		glog.V(4).Infof("Failed to get CanaryDeployment: %v", err)
		return
	}
	glog.V(4).Infof("Adding canary deployment %s", cd.Name)
	c.enqueueCanaryDeployment(cd)
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (c *CanaryDeploymentController) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *CanaryDeploymentController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncHandler(key.(string))
	c.handleErr(err, key)

	return true
}

func (c *CanaryDeploymentController) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < maxRetries {
		glog.V(2).Infof("Error syncing canary deployment %v: %v", key, err)
		c.queue.AddRateLimited(key)
		return
	}

	utilruntime.HandleError(err)
	glog.V(2).Infof("Dropping canary deployment %q out of the queue: %v", key, err)
	c.queue.Forget(key)
}

func (c *CanaryDeploymentController) enqueue(canaryDeployment *kanarini.CanaryDeployment) {
	key, err := controller.KeyFunc(canaryDeployment)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", canaryDeployment, err))
		return
	}

	c.queue.Add(key)
}

func (c *CanaryDeploymentController) enqueueRateLimited(canaryDeployment *kanarini.CanaryDeployment) {
	key, err := controller.KeyFunc(canaryDeployment)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", canaryDeployment, err))
		return
	}

	c.queue.AddRateLimited(key)
}

// enqueueAfter will enqueue a deployment after the provided amount of time.
func (c *CanaryDeploymentController) enqueueAfter(canaryDeployment *kanarini.CanaryDeployment, after time.Duration) {
	key, err := controller.KeyFunc(canaryDeployment)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", canaryDeployment, err))
		return
	}

	c.queue.AddAfter(key, after)
}

// syncDeployment will sync the deployment with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (c *CanaryDeploymentController) syncDeployment(key string) error {
	startTime := time.Now()
	glog.V(4).Infof("Started syncing canary deployment %q (%v)", key, startTime)
	defer func() {
		glog.V(4).Infof("Finished syncing canary deployment %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	canaryDeployment, err := c.cdLister.CanaryDeployments(namespace).Get(name)
	if errors.IsNotFound(err) {
		glog.V(2).Infof("Deployment %v has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	// Deep-copy otherwise we are mutating our cache.
	cd := canaryDeployment.DeepCopy()

	// List Deployments owned by this CanaryDeployment
	dList, err := c.getDeploymentsForCanaryDeployment(cd)
	if err != nil {
		return err
	}
	if cd.DeletionTimestamp != nil {
		return c.syncStatusOnly(cd, dList)
	}

	return c.rolloutCanary(cd, dList)
}

// getDeploymentsForCanaryDeployment returns the list of Deployments that this CanaryDeployment should manage.
func (c *CanaryDeploymentController) getDeploymentsForCanaryDeployment(cd *kanarini.CanaryDeployment) ([]*apps.Deployment, error) {
	// List all Deployments to find those we own
	dList, err := c.dLister.Deployments(cd.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var claimed []*apps.Deployment
	for _, d := range dList {
		controllerRef := metav1.GetControllerOf(d)
		if controllerRef == nil {
			// Orphaned. Ignore.
			continue
		}
		if controllerRef.UID != cd.GetUID() {
			// Owned by someone else. Ignore.
			continue
		}
		claimed = append(claimed, d)
	}
	return claimed, nil
}
