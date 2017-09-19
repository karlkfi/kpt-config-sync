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

// Package policynodewatcher is a utility for watching the policynode custom resource and reconnecting when the connection gets
// dropped by the server (which is by design).
package policynodewatcher

import (
	"sync"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventType defines the events returned by PolicyNodeWatcher
type EventType string

// Definitions for EventType constants
const (
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
)

// Convert a watch.EventType to EventType
func fromWatcherType(eventType watch.EventType) EventType {
	switch eventType {
	case watch.Added:
		return Added
	case watch.Modified:
		return Modified
	case watch.Deleted:
		return Deleted
	default:
		panic(errors.Errorf("watch.EventType value %s cannot be represented by EventType", eventType))
	}
}

// Interface defines the interface for the PolicyNodeWatcher.
type Interface interface {
	// Run starts the watcher, no callbacks on the EventHandler will be invoked before Run is called.
	Run(EventHandler)

	// ResourceVersion returns the resource version state which is initially set on construction
	// and then tracks the resource version at which new changes will be watched then finally
	// stores the last resource version on which something was seen when stopped.
	ResourceVersion() string

	// Stop instructs the watcher to stop.  This will tear down the watch, however, a callback may be
	// invoked after stop returns.
	Stop()

	// Wait waits for watching to terminate.  No callbacks on the EventHandler will be invoked after
	// Wait returns.  Calling Wait prior to stop is permissable as it will block until Stop is called.
	Wait()
}

// PolicyNodeWatcher handles wrapping the the watch method with a reconnect on timeout which happens
// periodically after establishing a watch on a kubernetes resource.
type PolicyNodeWatcher struct {
	policyHierarchyInterface policyhierarchy.Interface

	eventHandler         EventHandler
	resourceVersion      string
	resourceVersionMutex sync.Mutex

	// Constructs for stopping
	stop          chan struct{}
	stoppedAtomic int64

	// Constructs for waiting
	wait sync.WaitGroup
}

var _ Interface = &PolicyNodeWatcher{}

// New will create a new PolicyNodeWatcher from a ClientInterface, event handler and resourceVersion.
func New(
	policyHierarchyInterface policyhierarchy.Interface,
	resourceVersion string) *PolicyNodeWatcher {
	return &PolicyNodeWatcher{
		policyHierarchyInterface: policyHierarchyInterface,
		stop:            make(chan struct{}),
		resourceVersion: resourceVersion,
	}
}

// Run implements Interface
func (w *PolicyNodeWatcher) Run(eventHandler EventHandler) {
	if w.eventHandler != nil {
		panic(errors.Errorf("Event handler is already set"))
	}
	w.eventHandler = eventHandler
	w.wait.Add(1)
	go w.runInternal()
}

// runInternal handles wrapping the watch with a retry loop for resuming on watcher timeout.
func (w *PolicyNodeWatcher) runInternal() {
	glog.Infof("Starting PolicyNodeWatcher at resource version %s", w.ResourceVersion())
	for atomic.LoadInt64(&w.stoppedAtomic) == 0 {
		nextResourceVersion, err := w.watch()
		if err != nil {
			w.eventHandler.OnError(errors.Wrapf(
				err, "Error while reading from result channel"))
			w.Stop()
			break
		}

		w.setResourceVerision(nextResourceVersion)
	}
	w.wait.Done()
}

func (w *PolicyNodeWatcher) watch() (string, error) {
	nextResourceVersion := w.ResourceVersion()
	watchIface, err := w.policyHierarchyInterface.K8usV1().PolicyNodes().Watch(
		meta_v1.ListOptions{ResourceVersion: nextResourceVersion})

	if err != nil {
		return "", errors.Wrapf(err, "Failed to watch policy hierarchy")
	}

	for {
		select {
		case event, ok := <-watchIface.ResultChan():
			// invoke callback?  pass to channel?
			if !ok {
				glog.Infof("Client closed, exiting loop")
				return nextResourceVersion, nil
			}
			if event.Type == watch.Error {
				if event.Object == nil {
					return "", errors.Errorf("Got error event from watch result channel")
				}
				return "", errors.Errorf("Got error event from watch result channel with object: %#v", event.Object)
			}

			node := event.Object.(*policyhierarchy_v1.PolicyNode)
			nextResourceVersion = node.ResourceVersion
			w.eventHandler.OnEvent(fromWatcherType(event.Type), node)

		case _, ok := <-w.stop:
			if !ok {
				glog.Infof("Client got stop signal, stopping client poll")
				watchIface.Stop()
				return nextResourceVersion, nil
			}
		}
	}
}

// Stop implements Interface
func (w *PolicyNodeWatcher) Stop() {
	atomic.StoreInt64(&w.stoppedAtomic, 1)
	close(w.stop)
}

// Wait implements Interface
func (w *PolicyNodeWatcher) Wait() {
	w.wait.Wait()
}

// ResourceVersion implements Interface
func (w *PolicyNodeWatcher) ResourceVersion() string {
	w.resourceVersionMutex.Lock()
	defer w.resourceVersionMutex.Unlock()
	return w.resourceVersion
}

// setResourceVerision updates the resource version for the watcher.
func (w *PolicyNodeWatcher) setResourceVerision(resourceVersion string) {
	w.resourceVersionMutex.Lock()
	prevResourceVersion := w.resourceVersion
	w.resourceVersion = resourceVersion
	w.resourceVersionMutex.Unlock()
	glog.Infof("Advanced resourceVersion %s -> %s", prevResourceVersion, resourceVersion)
}
