package resource_quota

import (
	"flag"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/pkg/errors"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/util/workqueue"
)

// The Stolos Quota controller watches leaf native K8s quota and the policy node hierarchy and creates
// CRD StolosQuota objects in the non-leaf namespaces which contain the total usage of resources of all the
// descendant namespaces that are children of that policyspace.
type Controller struct {
	client    meta.Interface                  // Kubernetes/CRD client
	stopped   bool                            // Tracks internal state for stopping
	stopChan  chan struct{}                   // Channel will be closed when Stop() is called
	stopMutex sync.Mutex                      // Concurrency control for calling Stop()
	queue     workqueue.RateLimitingInterface // A work queue for items to be processed

	policyHierarchyInformerFactory externalversions.SharedInformerFactory // StolosInformers for policynodes and quota
	kubernetesInformerFactory      informers.SharedInformerFactory        // Core informers for native quota
}

// Number of times the worker will try to modify resource quota before giving up.
const WorkerNumRetries = 3

// How often to do a full non-leaf quota re-calculation.
var flagResyncPeriod = flag.Duration(
	"quota_resync_period", 30*time.Second, "The resync period for the quota controller")

func NewController(client meta.Interface, stopChan chan struct{}) *Controller {
	return &Controller{
		client:   client,
		stopChan: stopChan,
		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		policyHierarchyInformerFactory: externalversions.NewSharedInformerFactory(client.PolicyHierarchy(), *flagResyncPeriod),
		kubernetesInformerFactory:      informers.NewSharedInformerFactory(client.Kubernetes(), *flagResyncPeriod),
	}
}

func (c *Controller) Run() {
	c.runInformer()

	go c.runPeriodSync()
	go c.runWorker()
}

func (c *Controller) Stop() {
	glog.Infof("Stopping...")
	c.stopMutex.Lock()
	defer c.stopMutex.Unlock()
	if !c.stopped {
		c.stopped = true
		close(c.stopChan)
		c.queue.ShutDown()
	}
}

// Initializes informers and starts them to listen to changes.
func (c *Controller) runInformer() {
	// Get the informers to register them to start.
	c.policyHierarchyInformerFactory.K8us().V1().PolicyNodes().Informer()
	c.policyHierarchyInformerFactory.K8us().V1().StolosResourceQuotas().Informer()
	c.kubernetesInformerFactory.Core().V1().ResourceQuotas().Informer()

	// Start informer factories
	c.kubernetesInformerFactory.Start(c.stopChan)
	c.policyHierarchyInformerFactory.Start(c.stopChan)

	glog.Infof("Waiting for cache to sync...")
	c.kubernetesInformerFactory.WaitForCacheSync(nil)
	c.policyHierarchyInformerFactory.WaitForCacheSync(nil)
	glog.Infof("Caches synced.")
}

// Starts up the periodic full sync loop.
func (c *Controller) runPeriodSync() {
	ticker := time.NewTicker(*flagResyncPeriod)
	err := c.fullSync()
	if err != nil {
		glog.Errorf("Error during full resync due to %s", err)
	}
	for {
		select {
		case <-ticker.C:
			err := c.fullSync()
			if err != nil {
				glog.Errorf("Error during full resync due to %s", err)
				c.Stop()
				return
			}
		case <-c.stopChan:
			glog.Infof("Got stop channel close, exiting.")
			return
		}
	}
}

// Starts up the worker that processes items from the queue of writes to the api server.
func (c *Controller) runWorker() {
	// processAction will wait until an item becomes available.
	for c.processAction() {
	}
}

// Executes a full sync loading the full hierarchy of quotes and updates stolos quotes that may need updating.
func (c *Controller) fullSync() error {
	glog.Infof("Full sync")

	hierarchicalCache, err := NewHierarchicalQuotaCache(
		c.policyHierarchyInformerFactory.K8us().V1().PolicyNodes(),
		c.kubernetesInformerFactory.Core().V1().ResourceQuotas())
	if err != nil {
		return errors.Wrap(err, "Failed initializing cache during full sync")
	}
	stolosQuotaLister := c.policyHierarchyInformerFactory.K8us().V1().StolosResourceQuotas().Lister()
	for namespace, quotaNode := range hierarchicalCache.quotas {
		if quotaNode.policyspace {
			c.queue.Add(&UpsertStolosQuota{
				namespace: namespace,
				quotaSpec: v1.StolosResourceQuotaSpec{
					Status: quotaNode.quota.Status,
				},
				stolosQuotaLister:        stolosQuotaLister,
				policyHierarchiInterface: c.client.PolicyHierarchy(),
			})
		}
	}

	return nil
}

// processAction takes one item from the work queue and processes it.
// If no items are available queue.get() will wait for an item t become available.
// In the case of an error, it will retry the action up to WorkerNumRetries times.
// Will return false if ready to shutdown, true otherwise.
func (c *Controller) processAction() bool {
	actionItem, shutdown := c.queue.Get()
	if shutdown {
		glog.Infof("Shutting down Syncer queue processing worker")
		return false
	}
	defer c.queue.Done(actionItem)

	action := actionItem.(*UpsertStolosQuota)
	err := action.Execute()

	// All good!
	if err == nil {
		c.queue.Forget(actionItem)
		return true
	}

	// Drop forever
	if c.queue.NumRequeues(actionItem) > WorkerNumRetries {
		glog.Errorf("Discarding action %s due to %s", action.String(), err)
		c.queue.Forget(actionItem)
		return true
	}

	// Retry
	glog.Errorf("Error processing action %s due to %s, retrying", action.String(), err)
	c.queue.AddRateLimited(actionItem)
	return true
}
