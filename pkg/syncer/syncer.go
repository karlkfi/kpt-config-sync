/*
Copyright 2017 The Kubernetes Authors.
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
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	policynodelister_v1 "github.com/google/stolos/pkg/client/listers/k8us/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/google/stolos/pkg/util/policynode"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var flagResyncPeriod = flag.Duration(
	"resync_period", time.Minute, "The resync period for the syncer system")

var dryRun = flag.Bool(
	"dry_run", false, "Don't perform actions, just log what would have happened")

const WorkerNumRetries = 3

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
	client        meta.Interface                  // Kubernetes/CRD client
	errorCallback ErrorCallback                   // Callback invoked on error
	stopped       bool                            // Tracks internal state for stopping
	stopChan      chan struct{}                   // Channel will be closed when Stop() is called
	stopMutex     sync.Mutex                      // Concurrency control for calling Stop()
	syncers       []PolicyNodeSyncerInterface     // Syncers that this will call into on change events
	queue         workqueue.RateLimitingInterface // A work queue for items to be processed

	kubernetesInformerFactory      informers.SharedInformerFactory
	policyHierarchyInformerFactory externalversions.SharedInformerFactory
}

// New creates a new syncer that will use the given client interface
func New(client meta.Interface) *Syncer {
	kubernetesInformerFactory := informers.NewSharedInformerFactory(
		client.Kubernetes(), *flagResyncPeriod)
	policyHierarchyInformerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), *flagResyncPeriod)
	kubernetesCoreV1 := kubernetesInformerFactory.Core().V1()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return &Syncer{
		client:   client,
		stopChan: make(chan struct{}),
		syncers: []PolicyNodeSyncerInterface{
			// Namespace syncer must be first since quota depends on it
			NewNamespaceSyncer(client, kubernetesCoreV1.Namespaces().Lister(), queue),
			NewQuotaSyncer(client, kubernetesCoreV1.ResourceQuotas().Lister(), queue),
		},
		kubernetesInformerFactory:      kubernetesInformerFactory,
		policyHierarchyInformerFactory: policyHierarchyInformerFactory,
		queue: queue,
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
	policyNodesInformer := s.policyHierarchyInformerFactory.K8us().V1().PolicyNodes()
	informer := policyNodesInformer.Informer()
	lister := policyNodesInformer.Lister()

	// Subscribe to informer prior to starting the factories.
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd,
		UpdateFunc: s.onUpdate,
		DeleteFunc: s.onDelete,
	}
	informer.AddEventHandler(handler)

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

	go s.runResync(lister)
}

func (s *Syncer) runResync(lister policynodelister_v1.PolicyNodeLister) {
	ticker := time.NewTicker(*flagResyncPeriod)
	err := s.resync(lister)
	if err != nil {
		s.onError(err)
		return
	}
	for {
		select {
		case <-ticker.C:
			err := s.resync(lister)
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

func (s *Syncer) resync(lister policynodelister_v1.PolicyNodeLister) error {
	policyNodes, err := lister.List(labels.Everything())
	if err != nil {
		return errors.Wrapf(err, "Failed to list policy nodes")
	}
	for _, syncerInstance := range s.syncers {
		err := syncerInstance.PeriodicResync(policyNodes)
		if err != nil {
			return errors.Wrapf(err, "Failed to run periodic resync")
		}
	}
	return nil
}

// onAdd handles add events from the informer and de-duplicates the initial creates after the first
// list event.
func (s *Syncer) onAdd(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	resourceVersion := policynode.GetResourceVersionOrDie(policyNode)
	glog.V(1).Infof("onAdd %s (%d)", policyNode.Name, resourceVersion)

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.OnCreate(policyNode)
		if err != nil {
			s.onError(err)
			return
		}
	}
}

// onDelete handles delete events from the informer.
func (s *Syncer) onDelete(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	resourceVersion := policynode.GetResourceVersionOrDie(policyNode)
	glog.V(1).Infof("onDelete %s (%d)", policyNode.Name, resourceVersion)

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.OnDelete(policyNode)
		if err != nil {
			s.onError(err)
			return
		}
	}
}

// onUpdate handles update events from the informer
func (s *Syncer) onUpdate(oldObj, newObj interface{}) {
	oldPolicyNode := oldObj.(*policyhierarchy_v1.PolicyNode)
	newPolicyNode := newObj.(*policyhierarchy_v1.PolicyNode)
	glog.V(1).Infof(
		"onUpdate %s (%s->%s)", newPolicyNode.Name, oldPolicyNode.ResourceVersion, newPolicyNode.ResourceVersion)

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.OnUpdate(oldPolicyNode, newPolicyNode)
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
	defer s.queue.Done(actionItem)

	action := actionItem.(actions.Interface)
	if *dryRun {
		s.queue.Forget(actionItem)
		glog.Infof("Would have executed action %s", action.String())
		return true
	}
	err := action.Execute()

	// All good!
	if err == nil {
		s.queue.Forget(actionItem)
		return true
	}

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
