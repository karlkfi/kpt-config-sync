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

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/meta"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policynodewatcher"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/util/namespaceutil"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/util/set/stringset"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
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
	client            meta.Interface
	policyNodeWatcher policynodewatcher.Interface
	errorCallback     ErrorCallback
}

func New(client meta.Interface) *Syncer {
	return &Syncer{
		client: client,
	}
}

func (s *Syncer) Run(errorCallback ErrorCallback) {
	s.errorCallback = errorCallback
	go s.runInternal()
}

func (s *Syncer) Stop() {
	s.policyNodeWatcher.Stop()
}

func (s *Syncer) Wait() {
	s.policyNodeWatcher.Wait()
}

func (s *Syncer) computeActions(
	existingNamespaceList *core_v1.NamespaceList,
	policyNodeList *policyhierarchy_v1.PolicyNodeList) ([]client.NamespaceAction, error) {

	// Get the set of non-reserved, active namespaces
	existingNamespaces := stringset.New()
	terminatingNamespaces := stringset.New()
	for _, namespaceItem := range existingNamespaceList.Items {
		if namespaceutil.IsReserved(namespaceItem) {
			continue
		}
		switch namespaceItem.Status.Phase {
		case core_v1.NamespaceActive:
			existingNamespaces.Add(namespaceItem.Name)
		case core_v1.NamespaceTerminating:
			terminatingNamespaces.Add(namespaceItem.Name)
		}
	}

	declaredNamespaces := stringset.New()
	for _, policyNode := range policyNodeList.Items {
		declaredNamespaces.Add(policyNode.Spec.Name)
	}

	needsCreate := declaredNamespaces.Difference(existingNamespaces)
	needsDelete := existingNamespaces.Difference(declaredNamespaces)
	terminatingNeedsCreate := terminatingNamespaces.Intersection(needsCreate)

	if terminatingNeedsCreate.Size() != 0 {
		// TODO: add retry for this situation.
		return nil, errors.Errorf("Need to create namesapace %s which is currently terminating")
	}

	namespaceActions := []client.NamespaceAction{}
	needsCreate.ForEach(func(ns string) {
		namespaceActions = append(namespaceActions, client.NewNamespaceCreateAction(s.client.Kubernetes(), ns))
	})
	needsDelete.ForEach(func(ns string) {
		namespaceActions = append(namespaceActions, client.NewNamespaceDeleteAction(s.client.Kubernetes(), ns))
	})

	return namespaceActions, nil
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

	namespaceActions, err := s.computeActions(namespaceList, policyNodeList)
	if err != nil {
		return 0, err
	}

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

func (s *Syncer) runInternal() {
	resourceVersion, err := s.initialSync()
	if err != nil {
		s.errorCallback(errors.Wrapf(err, "Failed to perform initial sync"))
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
	s.errorCallback(errors.Wrapf(err, "Got error from PolicyNodeWatcher"))
}
