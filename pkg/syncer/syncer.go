/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package syncer

import (
	"flag"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/informers/externalversions"
	policynodelister_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/syncer/actions"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	flagResyncPeriod = flag.Duration(
		"resync_period", time.Minute, "The resync period for the syncer system")

	dryRun = flag.Bool(
		"dry_run", false, "Don't perform actions, just log what would have happened")
)

// WorkerNumRetries is the number of times an action will be retried in the work queue.
const WorkerNumRetries = 3

// Prometheus metrics
var (
	errTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total errors that occurred when executing syncer actions",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "error_total",
		},
		[]string{"namespace", "resource", "operation"},
	)
	eventTimes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Timestamps when syncer events occurred",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "event_timestamps",
		},
		[]string{"type"},
	)
	queueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current size of syncer action queue",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "queue_size",
		})
	syncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Syncer action duration distributions",
			Namespace: "nomos",
			Subsystem: "syncer",
			Name:      "action_duration_seconds",
			Buckets:   []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		},
		[]string{"namespace", "resource", "operation"},
	)
)

func init() {
	prometheus.MustRegister(errTotal)
	prometheus.MustRegister(eventTimes)
	prometheus.MustRegister(queueSize)
	prometheus.MustRegister(syncDuration)
}

// ErrorCallback is a callback which is called if Syncer encounters an error during execution
type ErrorCallback func(error)

// Interface is the interface for the namespace syncer.
type Interface interface {
	Run(ErrorCallback)
	Stop()
	Wait()
}

// Syncer implements the policy node syncer.  This will watch the policynodes then sync changes to
// namespaces and resource quotas.
type Syncer struct {
	client               meta.Interface                  // Kubernetes/CRD client
	errorCallback        ErrorCallback                   // Callback invoked on error
	stopped              bool                            // Tracks internal state for stopping
	stopChan             chan struct{}                   // Channel will be closed when Stop() is called
	stopMutex            sync.Mutex                      // Concurrency control for calling Stop()
	syncers              []PolicyNodeSyncerInterface     // Syncers that this will call into on change events
	queue                workqueue.RateLimitingInterface // A work queue for items to be processed
	clusterPolicySyncers []ClusterPolicySyncerInterface  // ClusterPolicy syncers

	kubernetesInformerFactory      informers.SharedInformerFactory
	policyHierarchyInformerFactory externalversions.SharedInformerFactory

	policyNodeLister policynodelister_v1.PolicyNodeLister
}

// New creates a new syncer that will use the given client interface
func New(client meta.Interface) *Syncer {
	kubernetesInformerFactory := informers.NewSharedInformerFactory(
		client.Kubernetes(), *flagResyncPeriod)
	policyHierarchyInformerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), *flagResyncPeriod)
	kubernetesCoreV1 := kubernetesInformerFactory.Core().V1()
	rbacV1 := kubernetesInformerFactory.Rbac().V1()
	policyHierarchyV1 := policyHierarchyInformerFactory.Nomos().V1()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	syncers := []PolicyNodeSyncerInterface{
		// Namespace syncer must be first since quota depends on it
		NewNamespaceSyncer(client, kubernetesCoreV1.Namespaces().Lister(), queue),
		NewQuotaSyncer(client, kubernetesCoreV1.ResourceQuotas(), policyHierarchyV1.PolicyNodes(), queue),
		NewFlatteningSyncer(
			queue, actions.NewRoleBindingResource(
				client.Kubernetes(), rbacV1.RoleBindings().Lister()),
			actions.NewRoleResource(
				client.Kubernetes(), rbacV1.Roles().Lister()),
		),
	}

	clusterPolicySyncers := []ClusterPolicySyncerInterface{
		NewClusterRoleSyncer(client, rbacV1.ClusterRoles().Lister(), queue),
	}

	return &Syncer{
		client:                         client,
		stopChan:                       make(chan struct{}),
		syncers:                        syncers,
		kubernetesInformerFactory:      kubernetesInformerFactory,
		policyHierarchyInformerFactory: policyHierarchyInformerFactory,
		queue:                queue,
		clusterPolicySyncers: clusterPolicySyncers,
	}
}

// Run starts the syncer, any errors encountered will be returned through the error
// callback
func (s *Syncer) Run(errorCallback ErrorCallback) {
	s.errorCallback = errorCallback

	go s.runInformer()
	go s.runWorker()
}

// Stop asynchronously instructs the syncer to stop.
func (s *Syncer) Stop() {
	s.stopMutex.Lock()
	defer s.stopMutex.Unlock()
	if !s.stopped {
		s.stopped = true
		close(s.stopChan)
		s.queue.ShutDown()
	}
}

// Wait will wait for the syncer to complete then exit
func (s *Syncer) Wait() {
}

func (s *Syncer) runInformer() {
	policyNodes := s.policyHierarchyInformerFactory.Nomos().V1().PolicyNodes()
	s.policyNodeLister = policyNodes.Lister()

	// Subscribe to informer prior to starting the factories.
	policyNodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.addPolicyNode,
		UpdateFunc: s.updatePolicyNode,
		DeleteFunc: s.deletePolicyNode,
	})

	clusterPolicies := s.policyHierarchyInformerFactory.Nomos().V1().ClusterPolicies()
	clusterPolicies.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.addClusterPolicy,
		UpdateFunc: s.updateClusterPolicy,
		DeleteFunc: s.deleteClusterPolicy,
	})

	// Start informer factories
	s.kubernetesInformerFactory.Start(s.stopChan)
	s.policyHierarchyInformerFactory.Start(s.stopChan)

	glog.Infof("Waiting for cache to sync...")
	kubernetesSyncTypes := s.kubernetesInformerFactory.WaitForCacheSync(s.stopChan)
	policyHierarchySyncTypes := s.policyHierarchyInformerFactory.WaitForCacheSync(s.stopChan)
	for _, syncTypes := range []map[reflect.Type]bool{kubernetesSyncTypes, policyHierarchySyncTypes} {
		for syncType, ok := range syncTypes {
			if !ok {
				elemType := syncType.Elem()
				glog.Errorf("Failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
				s.Stop()
				return
			}
		}
	}
	glog.Infof("Caches synced.")

	go s.runResync(s.policyNodeResync)
}

func (s *Syncer) runResync(resync func() error) {
	ticker := time.NewTicker(*flagResyncPeriod)
	err := resync()
	if err != nil {
		s.onError(err)
		return
	}
	for {
		select {
		case <-ticker.C:
			err := resync()
			if err != nil {
				s.onError(err)
				return
			}
		case <-s.stopChan:
			glog.V(1).Infof("Got stop channel close, exiting.")
			return
		}
	}
}

func (s *Syncer) policyNodeResync() error {
	policyNodes, err := s.policyNodeLister.List(labels.Everything())
	if err != nil {
		return errors.Wrapf(err, "Failed to list policy nodes")
	}
	glog.V(1).Infof("PolicyNode Periodic Resync")
	eventTimes.WithLabelValues("resync").Set(float64(time.Now().Unix()))

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.PeriodicResync(policyNodes)
		if err != nil {
			return errors.Wrapf(err, "Failed to run periodic resync")
		}
	}
	return nil
}

// addPolicyNode handles add events from the informer and de-duplicates the initial creates after the first
// list event.
func (s *Syncer) addPolicyNode(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	glog.V(1).Infof("addPolicyNode %s (%s)", policyNode.Name, policyNode.ResourceVersion)
	eventTimes.WithLabelValues("add").Set(float64(time.Now().Unix()))

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.OnCreate(policyNode)
		if err != nil {
			s.onError(err)
			return
		}
	}
	queueSize.Set(float64(s.queue.Len()))
}

// deletePolicyNode handles delete events from the informer.
func (s *Syncer) deletePolicyNode(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	glog.V(1).Infof("deletePolicyNode %s (%d)", policyNode.Name, policyNode.ResourceVersion)
	eventTimes.WithLabelValues("delete").Set(float64(time.Now().Unix()))

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.OnDelete(policyNode)
		if err != nil {
			s.onError(err)
			return
		}
	}
	queueSize.Set(float64(s.queue.Len()))
}

// updatePolicyNode handles update events from the informer
func (s *Syncer) updatePolicyNode(oldObj, newObj interface{}) {
	oldPolicyNode := oldObj.(*policyhierarchy_v1.PolicyNode)
	newPolicyNode := newObj.(*policyhierarchy_v1.PolicyNode)
	glog.V(1).Infof(
		"updatePolicyNode %s (%s->%s)", newPolicyNode.Name, oldPolicyNode.ResourceVersion, newPolicyNode.ResourceVersion)
	eventTimes.WithLabelValues("update").Set(float64(time.Now().Unix()))

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.OnUpdate(oldPolicyNode, newPolicyNode)
		if err != nil {
			s.onError(err)
			return
		}
	}
	queueSize.Set(float64(s.queue.Len()))
}

func (s *Syncer) addClusterPolicy(obj interface{}) {
	clusterPolicy := obj.(*policyhierarchy_v1.ClusterPolicy)
	glog.V(1).Infof(
		"addClusterPolicy %s (%s)", clusterPolicy.Name, clusterPolicy.ResourceVersion)

	for _, handler := range s.clusterPolicySyncers {
		err := handler.OnCreate(clusterPolicy)
		if err != nil {
			s.onError(err)
			return
		}
	}
}

func (s *Syncer) updateClusterPolicy(oldObj, newObj interface{}) {
	oldClusterPolicy := oldObj.(*policyhierarchy_v1.ClusterPolicy)
	clusterPolicy := newObj.(*policyhierarchy_v1.ClusterPolicy)
	glog.V(1).Infof(
		"updateClusterPolicy %s (%s->%s)",
		clusterPolicy.Name, oldClusterPolicy.ResourceVersion, clusterPolicy.ResourceVersion)

	for _, handler := range s.clusterPolicySyncers {
		err := handler.OnUpdate(oldClusterPolicy, clusterPolicy)
		if err != nil {
			s.onError(err)
			return
		}
	}
}

func (s *Syncer) deleteClusterPolicy(obj interface{}) {
	clusterPolicy := obj.(*policyhierarchy_v1.ClusterPolicy)
	glog.V(1).Infof(
		"deleteClusterPolicy %s (%s)", clusterPolicy.Name, clusterPolicy.ResourceVersion)

	for _, handler := range s.clusterPolicySyncers {
		err := handler.OnDelete(clusterPolicy)
		if err != nil {
			s.onError(err)
			return
		}
	}
}

func (s *Syncer) onError(err error) {
	s.errorCallback(err)
	s.Stop()
}

func (s *Syncer) runWorker() {
	for s.processAction() {
	}
}

// processAction takes one item from the work queue and processes it.
// In the case of an error, it will retry the action up to 3 times.
// Will return false if ready to shutdown, true otherwise.
func (s *Syncer) processAction() bool {
	actionItem, shutdown := s.queue.Get()
	if shutdown {
		glog.Infof("Shutting down Syncer queue processing worker")
		return false
	}
	defer queueSize.Set(float64(s.queue.Len()))
	defer s.queue.Done(actionItem)

	action := actionItem.(action.Interface)
	if *dryRun {
		s.queue.Forget(actionItem)
		glog.Infof("Would have executed action %s", action.String())
		return true
	}
	exTimer := prometheus.NewTimer(syncDuration.WithLabelValues(action.Namespace(), action.Resource(), string(action.Operation())))
	err := action.Execute()
	exTimer.ObserveDuration()

	// All good!
	if err == nil {
		s.queue.Forget(actionItem)
		return true
	}
	errTotal.WithLabelValues(action.Namespace(), action.Resource(), string(action.Operation())).Inc()

	// Drop forever
	if s.queue.NumRequeues(actionItem) > WorkerNumRetries {
		glog.Errorf("Discarding action %s due to %s", action.String(), err)
		s.queue.Forget(actionItem)
		return true
	}

	// Retry
	glog.Errorf("Error processing action %s due to %s, retrying", action.String(), err)
	s.queue.AddRateLimited(actionItem)
	return true
}
