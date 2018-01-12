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
	"github.com/prometheus/client_golang/prometheus"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
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
	quotaCache                     *HierarchicalQuotaCache                // A cache of quotas to use between full re-syncs
}

// Number of times the worker will try to modify resource quota before giving up.
const WorkerNumRetries = 3

// How often to do a full non-leaf quota re-calculation.
var flagResyncPeriod = flag.Duration(
	"quota_resync_period", 30*time.Second, "The resync period for the quota controller")

// Prometheus metrics
var (
	errTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total errors that occurred when executing quota actions",
			Namespace: "stolos",
			Subsystem: "quota",
			Name:      "error_total",
		},
		[]string{"namespace"},
	)
	eventTimes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Timestamps when quota events occurred",
			Namespace: "stolos",
			Subsystem: "quota",
			Name:      "event_timestamps",
		},
		[]string{"type"},
	)
	queueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current size of quota action queue",
			Namespace: "stolos",
			Subsystem: "quota",
			Name:      "queue_size",
		})
	syncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Quota action duration distributions",
			Namespace: "stolos",
			Subsystem: "quota",
			Name:      "action_duration_seconds",
			Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"namespace"},
	)
)

func init() {
	prometheus.MustRegister(errTotal)
	prometheus.MustRegister(eventTimes)
	prometheus.MustRegister(queueSize)
	prometheus.MustRegister(syncDuration)
}

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
	c.policyHierarchyInformerFactory.Stolos().V1().PolicyNodes().Informer()
	c.policyHierarchyInformerFactory.Stolos().V1().StolosResourceQuotas().Informer()
	c.kubernetesInformerFactory.Core().V1().ResourceQuotas().Informer()

	// Start informer factories
	c.kubernetesInformerFactory.Start(c.stopChan)
	c.policyHierarchyInformerFactory.Start(c.stopChan)

	glog.Infof("Waiting for cache to sync...")
	c.kubernetesInformerFactory.WaitForCacheSync(nil)
	c.policyHierarchyInformerFactory.WaitForCacheSync(nil)
	glog.Infof("Caches synced.")

	// Add handler for Quotas
	quotaHandler := cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.onQuotaUpdate,
		// Add and Delete imply changes in the PolicyNode hierarchy and should be handled
		// by a watch on the policy node instead.
	}
	c.kubernetesInformerFactory.Core().V1().ResourceQuotas().Informer().AddEventHandler(quotaHandler)
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
	eventTimes.WithLabelValues("resync").Set(float64(time.Now().Unix()))

	hierarchicalCache, err := NewHierarchicalQuotaCache(
		c.policyHierarchyInformerFactory.Stolos().V1().PolicyNodes(),
		c.kubernetesInformerFactory.Core().V1().ResourceQuotas())
	if err != nil {
		return errors.Wrap(err, "Failed initializing cache during full sync")
	}
	stolosQuotaLister := c.policyHierarchyInformerFactory.Stolos().V1().StolosResourceQuotas().Lister()
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
	queueSize.Set(float64(c.queue.Len()))

	c.quotaCache = hierarchicalCache
	return nil
}

func (c *Controller) onQuotaUpdate(oldObj, newObj interface{}) {
	if c.quotaCache == nil {
		return // Not initialized yet, we will let full sync handle things.
	}

	newQuota := newObj.(*core_v1.ResourceQuota)
	if newQuota.Name != ResourceQuotaObjectName || newQuota.Labels[NamespaceTypeLabel] != NamespaceTypeWorkload {
		return // Some other quota object which we don't care about.
	}

	stolosQuotaNamespacesToUpdate, err := c.quotaCache.UpdateLeaf(*newQuota)
	if err != nil {
		// This can happen when the cache is malformed, defer to full sync.
		glog.Infof("Failed on quota update event for namespace %q due to %s. Ignoring update.",
			newQuota.Namespace, err)
	}
	eventTimes.WithLabelValues("update").Set(float64(time.Now().Unix()))

	stolosQuotaLister := c.policyHierarchyInformerFactory.Stolos().V1().StolosResourceQuotas().Lister()
	for _, namespace := range stolosQuotaNamespacesToUpdate {
		c.queue.Add(&UpsertStolosQuota{
			namespace: namespace,
			quotaSpec: v1.StolosResourceQuotaSpec{
				Status: c.quotaCache.quotas[namespace].quota.Status,
			},
			stolosQuotaLister:        stolosQuotaLister,
			policyHierarchiInterface: c.client.PolicyHierarchy(),
		})
	}
	queueSize.Set(float64(c.queue.Len()))
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
	defer queueSize.Set(float64(c.queue.Len()))
	defer c.queue.Done(actionItem)

	action := actionItem.(*UpsertStolosQuota)
	exTimer := prometheus.NewTimer(syncDuration.WithLabelValues(action.Namespace()))
	err := action.Execute()
	exTimer.ObserveDuration()

	// All good!
	if err == nil {
		c.queue.Forget(actionItem)
		return true
	}
	errTotal.WithLabelValues(action.Namespace()).Inc()

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
