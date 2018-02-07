/*
Copyright 2017 The Stolos Authors.
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

package remotecluster

import (
	"flag"
	"reflect"
	"sync"
	"time"

	"fmt"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	policynodelister_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/syncer"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/google/stolos/pkg/util/policynode"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var flagResyncPeriod = flag.Duration(
	"pnc_resync_period", 15*time.Minute, "Resync period for policy node controller ")

var dryRun = flag.Bool(
	"pnc_dry_run", false, "Don't perform actions, just log what would have happened")

// workerNumRetries is the number of times an action will be retried in the work queue.
const workerNumRetries = 3

// Controller implements the policy node controller.  This will watch the PolicyNodes(s) on remote cluster
// and sync changes to corresponding resources on local cluster.
type Controller struct {
	remoteClient meta.Interface                   // Client for remote Kubernetes cluster API server
	localClient  meta.Interface                   // Client for local Kubernetes cluster API server
	stopped      bool                             // Tracks internal state for stopping
	stopChan     chan struct{}                    // Channel will be closed when Stop() is called
	stopMutex    sync.Mutex                       // Concurrency control for calling Stop()
	copier       syncer.PolicyNodeSyncerInterface // Syncer that this will call into on change events
	queue        workqueue.RateLimitingInterface  // A work queue for items to be processed

	remoteInformerFactory externalversions.SharedInformerFactory
	localInformerFactory  externalversions.SharedInformerFactory
}

// NewController creates a new policy node controller.
func NewController(localClient meta.Interface, remoteClient meta.Interface, stopChan chan struct{}) *Controller {
	remoteInformerFactory := externalversions.NewSharedInformerFactory(
		remoteClient.PolicyHierarchy(), *flagResyncPeriod)
	localInformerFactory := externalversions.NewSharedInformerFactory(
		localClient.PolicyHierarchy(), *flagResyncPeriod)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	return &Controller{
		remoteClient: remoteClient,
		localClient:  localClient,
		stopChan:     stopChan,
		queue:        queue,
		remoteInformerFactory: remoteInformerFactory,
		localInformerFactory:  localInformerFactory,
	}
}

// Run starts the controller.
func (p *Controller) Run() {
	p.runInformer()
	go p.runWorker()
}

// Stop asynchronously instructs the controller to stop.
func (p *Controller) Stop() {
	p.stopMutex.Lock()
	defer p.stopMutex.Unlock()
	if !p.stopped {
		p.stopped = true
		close(p.stopChan)
		p.queue.ShutDown()
	}
}

func (p *Controller) runInformer() {
	remoteInformer := p.remoteInformerFactory.Stolos().V1().PolicyNodes()
	localInformer := p.localInformerFactory.Stolos().V1().PolicyNodes()

	p.copier = NewPolicyNodeCopier(localInformer.Lister(), p.localClient.PolicyHierarchy().StolosV1().PolicyNodes(), p.queue)

	// Subscribe to informer prior to starting the factories.
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    p.onAdd,
		UpdateFunc: p.onUpdate,
		DeleteFunc: p.onDelete,
	}
	remoteInformer.Informer().AddEventHandler(handler)

	// Start informer factories
	p.remoteInformerFactory.Start(p.stopChan)
	p.localInformerFactory.Start(p.stopChan)

	glog.Infof("Waiting for cache to sync...")
	remoteSynced := p.remoteInformerFactory.WaitForCacheSync(p.stopChan)
	localSynced := p.localInformerFactory.WaitForCacheSync(p.stopChan)

	for _, syncTypes := range []map[reflect.Type]bool{remoteSynced, localSynced} {
		for syncType, ok := range syncTypes {
			if !ok {
				elemType := syncType.Elem()
				glog.Errorf("Failed to sync %s:%s", elemType.PkgPath(), elemType.Name())
				p.Stop()
				return
			}
		}
	}
	glog.Infof("Caches synced.")

	go p.runResync(remoteInformer.Lister())
}

func (p *Controller) runResync(lister policynodelister_v1.PolicyNodeLister) {
	ticker := time.NewTicker(*flagResyncPeriod)
	err := p.resync(lister)
	if err != nil {
		p.onError(err)
		return
	}
	for {
		select {
		case <-ticker.C:
			err := p.resync(lister)
			if err != nil {
				p.onError(err)
				return
			}
		case <-p.stopChan:
			glog.Infof("Got stop channel close, exiting.")
			return
		}
	}
}

func (p *Controller) resync(lister policynodelister_v1.PolicyNodeLister) error {
	policyNodes, err := lister.List(labels.Everything())
	if err != nil {
		return errors.Wrapf(err, "Failed to list policy nodes")
	}
	glog.Infof("Periodic Resync")
	err = p.copier.PeriodicResync(policyNodes)
	if err != nil {
		return errors.Wrapf(err, "Failed to run periodic resync")
	}
	return nil
}

// onAdd handles add events from the informer and de-duplicates the initial creates after the first
// list event.
func (p *Controller) onAdd(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	resourceVersion := policynode.GetResourceVersionOrDie(policyNode)
	glog.Infof("onAdd %q (%d)", policyNode.Name, resourceVersion)

	err := p.copier.OnCreate(policyNode)
	if err != nil {
		p.onError(err)
		return
	}
}

// onDelete handles delete events from the informer.
func (p *Controller) onDelete(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	resourceVersion := policynode.GetResourceVersionOrDie(policyNode)
	glog.Infof("onDelete %q (%d)", policyNode.Name, resourceVersion)

	err := p.copier.OnDelete(policyNode)
	if err != nil {
		p.onError(err)
		return
	}
}

// onUpdate handles update events from the informer
func (p *Controller) onUpdate(oldObj, newObj interface{}) {
	oldPolicyNode := oldObj.(*policyhierarchy_v1.PolicyNode)
	newPolicyNode := newObj.(*policyhierarchy_v1.PolicyNode)
	glog.Infof(
		"onUpdate %q (%s->%s)", newPolicyNode.Name, oldPolicyNode.ResourceVersion, newPolicyNode.ResourceVersion)

	err := p.copier.OnUpdate(oldPolicyNode, newPolicyNode)
	if err != nil {
		p.onError(err)
		return
	}
}

func (p *Controller) onError(err error) {
	glog.Error(err)
	p.Stop()
}

func (p *Controller) runWorker() {
	for p.processAction() {
	}
}

// processAction takes one item from the work queue and processes it.
// In the case of an error, it will retry the action up to 3 times.
// Will return false if ready to shutdown, true otherwise.
func (p *Controller) processAction() bool {
	actionItem, shutdown := p.queue.Get()
	if shutdown {
		glog.Infof("Shutting down Controller queue processing worker")
		return false
	}
	defer p.queue.Done(actionItem)

	action := actionItem.(actions.Interface)
	if *dryRun {
		p.queue.Forget(actionItem)
		glog.Infof("Would have executed action %s", action.String())
		return true
	}
	err := action.Execute()

	// Action succeeded
	if err == nil {
		p.queue.Forget(actionItem)
		return true
	}

	// Action failed and exceeded retries
	if p.queue.NumRequeues(actionItem) > workerNumRetries {
		glog.Errorf("Discarding action %s due to %s", action.String(), err)
		p.queue.Forget(actionItem)
		return true
	}

	// There was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring
	err = fmt.Errorf("Error processing action %s: %v, retyring", action.String(), err)
	utilruntime.HandleError(err)

	// Retry
	glog.Error(err)
	p.queue.AddRateLimited(actionItem)
	return true
}
