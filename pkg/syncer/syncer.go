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
	k8us_v1 "github.com/google/stolos/pkg/client/listers/k8us/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/util/policynode"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

var dryRun = flag.Bool(
	"dry_run", false, "Don't perform actions, just log what would have happened")

// ErrorCallback is a callback which is called if Syncer encounters an error during execution
type ErrorCallback func(error)

// Interface is the interface for the namespace syncer.
type Interface interface {
	Run(ErrorCallback)
	Stop()
	Wait()
}

// Syncer implements the namespace syncer.  This will watch the policynodes then sync changes to
// namespaces.
type Syncer struct {
	client          meta.Interface              // Kubernetes/CRD client
	errorCallback   ErrorCallback               // Callback invoked on error
	stopped         bool                        // Tracks internal state for stopping
	stopChan        chan struct{}               // Channel will be closed when Stop() is called
	stopMutex       sync.Mutex                  // Concurrency control for calling Stop()
	resourceVersion int64                       // The highest resource version we have seen for a PollicyNode
	syncers         []PolicyNodeSyncerInterface // Syncers that this will call into on change events

	kubernetesInformerFactory      informers.SharedInformerFactory
	policyHierarchyInformerFactory externalversions.SharedInformerFactory
}

// New creates a new syncer that will use the given client interface
func New(client meta.Interface) *Syncer {
	kubernetesInformerFactory := informers.NewSharedInformerFactory(
		client.Kubernetes(), time.Minute)
	policyHierarchyInformerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), time.Minute)
	kubernetesCoreV1 := kubernetesInformerFactory.Core().V1()
	return &Syncer{
		client:   client,
		stopChan: make(chan struct{}),
		syncers: []PolicyNodeSyncerInterface{
			// Namespace syncer must be first since quota depends on it
			NewNamespaceSyncer(client, kubernetesCoreV1.Namespaces().Lister()),
			NewQuotaSyncer(client, kubernetesCoreV1.ResourceQuotas().Lister()),
		},
		kubernetesInformerFactory:      kubernetesInformerFactory,
		policyHierarchyInformerFactory: policyHierarchyInformerFactory,
	}
}

// Run starts the syncer, any errors encountered will be returned through the error
// callback
func (s *Syncer) Run(errorCallback ErrorCallback) {
	s.errorCallback = errorCallback
	go s.runInternal()
}

// Stop asynchronously instructs the syncer to stop.
func (s *Syncer) Stop() {
	s.stopMutex.Lock()
	defer s.stopMutex.Unlock()
	if !s.stopped {
		s.stopped = true
		close(s.stopChan)
	}
}

// Wait will wait for the syncer to complete then exit
func (s *Syncer) Wait() {
	// TODO: figure out if we need a wait function.
}

func (s *Syncer) initialSync(lister k8us_v1.PolicyNodeLister) error {
	nodes, err := lister.List(labels.Everything())
	if err != nil {
		return err
	}

	// Set resource version
	for _, node := range nodes {
		resourceVersion, err := policynode.GetResourceVersion(node)
		if err != nil {
			return errors.Wrapf(err, "Failed to get resource version from %#v", node)
		}
		if s.resourceVersion < resourceVersion {
			s.resourceVersion = resourceVersion
		}
	}

	for _, syncerInstance := range s.syncers {
		err := syncerInstance.InitialSync(nodes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Syncer) runInternal() {
	policyNodesInformer := s.policyHierarchyInformerFactory.K8us().V1().PolicyNodes()
	informer := policyNodesInformer.Informer()
	lister := policyNodesInformer.Lister()

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
				return
			}
		}
	}

	err := s.initialSync(lister)
	if err != nil {
		s.onError(err)
		return
	}

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd,
		UpdateFunc: s.onUpdate,
		DeleteFunc: s.onDelete,
	}
	informer.AddEventHandler(handler)
}

// onAdd handles add events from the informer and de-duplicates the initial creates after the first
// list event.
func (s *Syncer) onAdd(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	resourceVersion := policynode.GetResourceVersionOrDie(policyNode)
	glog.V(1).Infof("onAdd %s (%d)", policyNode.Name, resourceVersion)
	if resourceVersion <= s.resourceVersion {
		glog.V(2).Infof("suppressed onAdd %s (%s <= %d)", policyNode.Name, policyNode.ResourceVersion, s.resourceVersion)
		return
	}
	s.resourceVersion = resourceVersion

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
	if oldPolicyNode.ResourceVersion == newPolicyNode.ResourceVersion {
		glog.V(2).Infof("SUPPRESSED: onUpdate due to same resource version %s (%d)", oldPolicyNode.Name, oldPolicyNode.ResourceVersion)
		return
	}

	newResourceVersion := policynode.GetResourceVersionOrDie(newPolicyNode)
	if newResourceVersion <= s.resourceVersion {
		panic(errors.Errorf("Unexpected resource version replay at %d: %#v", s.resourceVersion, newPolicyNode))
		return
	}

	glog.V(1).Infof("onUpdate %s (%s) -> (%s)", oldPolicyNode.Name, oldPolicyNode.ResourceVersion, newPolicyNode.ResourceVersion)
	s.resourceVersion = newResourceVersion

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
