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
	"strconv"
	"sync"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/policynodewatcher"
	"github.com/google/stolos/pkg/util/namespaceutil"
	"github.com/google/stolos/pkg/util/set/stringset"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	client                meta.Interface
	errorCallback         ErrorCallback
	policyNodeWatcher     policynodewatcher.Interface
	stopped               bool
	policyNodeWatcherLock sync.Mutex // Guards policyNodeWatcher and stopped
}

// New creates a new syncer that will use the given client interface
func New(client meta.Interface) *Syncer {
	return &Syncer{
		client: client,
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
	s.policyNodeWatcherLock.Lock()
	defer s.policyNodeWatcherLock.Unlock()
	s.stopped = true
	if s.policyNodeWatcher != nil {
		s.policyNodeWatcher.Stop()
	}
}

// Wait will wait for the syncer to complete then exit
func (s *Syncer) Wait() {
	s.policyNodeWatcherLock.Lock()
	defer s.policyNodeWatcherLock.Unlock()
	if s.policyNodeWatcher != nil {
		s.policyNodeWatcher.Wait()
	}
}

func (s *Syncer) computeActions(
	existingNamespaceList *core_v1.NamespaceList,
	policyNodeList *policyhierarchy_v1.PolicyNodeList) []client.NamespaceAction {

	// Get the set of non-reserved, active namespaces
	existingNamespaces := stringset.New()
	for _, namespaceItem := range existingNamespaceList.Items {
		if namespaceutil.IsReserved(namespaceItem) {
			continue
		}
		switch namespaceItem.Status.Phase {
		case core_v1.NamespaceActive:
			existingNamespaces.Add(namespaceItem.Name)
		case core_v1.NamespaceTerminating:
		}
	}

	declaredNamespaces := stringset.New()
	for _, policyNode := range policyNodeList.Items {
		declaredNamespaces.Add(policyNode.ObjectMeta.Name)
	}

	needsCreate := declaredNamespaces.Difference(existingNamespaces)
	needsDelete := existingNamespaces.Difference(declaredNamespaces)

	namespaceActions := []client.NamespaceAction{}
	needsCreate.ForEach(func(ns string) {
		glog.Infof("Adding create operation for %s", ns)
		namespaceActions = append(namespaceActions, client.NewNamespaceCreateAction(s.client.Kubernetes(), ns))
	})
	needsDelete.ForEach(func(ns string) {
		glog.Infof("Adding delete operation for %s", ns)
		namespaceActions = append(namespaceActions, client.NewNamespaceDeleteAction(s.client.Kubernetes(), ns))
	})

	return namespaceActions
}

func (s *Syncer) initialSync() (int64, error) {
	glog.Info("Performing initial sync on namespaces")
	namespaceList, err := s.client.Kubernetes().CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return 0, err
	}

	namespaceResourceVersion, err := strconv.ParseInt(namespaceList.ResourceVersion, 10, 64)
	if err != nil {
		return 0, err
	}

	policyNodeList, err := s.client.PolicyHierarchy().K8usV1().PolicyNodes().List(meta_v1.ListOptions{})
	if err != nil {
		return 0, err
	}
	policyNodeResourceVersion, err := strconv.ParseInt(namespaceList.ResourceVersion, 10, 64)
	if err != nil {
		return 0, err
	}
	glog.Infof(
		"Listed namespaces at resource version %d, policy nodes at %d",
		namespaceResourceVersion, policyNodeResourceVersion)

	namespaceActions := s.computeActions(namespaceList, policyNodeList)
	for _, action := range namespaceActions {
		err := action.Execute()
		if err != nil {
			glog.Infof("Action %s %s failed due to %s", action.Name(), action.Operation(), err)
			return 0, err
		}
	}

	glog.Infof("Finished initial sync, will use resource version %s", namespaceList.ResourceVersion)
	return policyNodeResourceVersion, nil
}

// isStopped returns if the syncer is stopped
func (s *Syncer) isStopped() bool {
	s.policyNodeWatcherLock.Lock()
	defer s.policyNodeWatcherLock.Unlock()
	return s.stopped
}

func (s *Syncer) runInternal() {
	if s.isStopped() {
		glog.Info("Syncer stopped, exiting.")
		return
	}

	resourceVersion, err := s.initialSync()
	if err != nil {
		s.errorCallback(errors.Wrapf(err, "Failed to perform initial sync"))
		return
	}

	s.policyNodeWatcherLock.Lock()
	defer s.policyNodeWatcherLock.Unlock()
	if s.stopped {
		glog.Info("Syncer exiting loop")
		return
	}
	s.policyNodeWatcher = policynodewatcher.New(s.client.PolicyHierarchy(), resourceVersion)
	s.policyNodeWatcher.Run(policynodewatcher.NewEventHandler(s.onPolicyNodeEvent, s.onPolicyNodeError))
}

func (s *Syncer) getEventAction(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) client.NamespaceAction {
	var action client.NamespaceAction
	namespace := policyNode.Name
	glog.V(3).Infof("Got event %s namespace %s resourceVersion %s", eventType, policyNode.Name, policyNode.ResourceVersion)
	switch eventType {
	case policynodewatcher.Added:
		action = client.NewNamespaceCreateAction(s.client.Kubernetes(), namespace)
	case policynodewatcher.Deleted:
		action = client.NewNamespaceDeleteAction(s.client.Kubernetes(), namespace)
	case policynodewatcher.Modified:
		glog.Info("Got modified event for %s, ignoring", policyNode.Name)
	}
	return action
}

func (s *Syncer) onPolicyNodeEvent(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	// NOTE: There is a bug here where the namespace create action can operate on a namespace that is presently
	// terminating.  This needs to undersand when the namespace is finally deleted then attempt creation, preferably
	// with some sort of timeout.
	action := s.getEventAction(eventType, policyNode)
	if action == nil {
		return
	}

	err := action.Execute()
	if err != nil {
		s.errorCallback(errors.Wrapf(
			err, "Failed to perform action %s on %s: %s", action.Operation(), action.Name(), err))
		return
	}
}

func (s *Syncer) onPolicyNodeError(err error) {
	if cause := errors.Cause(err); api_errors.IsGone(cause) {
		glog.Infof("Got IsGone error, restarting sync: %s", cause)
		s.runInternal()
		return
	}

	s.errorCallback(errors.Wrapf(err, "Got error from PolicyNodeWatcher"))
}
