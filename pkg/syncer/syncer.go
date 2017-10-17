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
	"k8s.io/apimachinery/pkg/fields"
	"reflect"
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

func (s *Syncer) computeNamespaceActions(policyNodeList *policyhierarchy_v1.PolicyNodeList) ([]client.NamespaceAction, error) {

	existingNamespaceList, err := s.client.Kubernetes().CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return s.computeNamespaceActionsWithNamespaceList(existingNamespaceList, policyNodeList), nil
}

func (s *Syncer) computeNamespaceActionsWithNamespaceList(existingNamespaceList *core_v1.NamespaceList,
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

func (s *Syncer) computeResourceQuotaActions(policyNodeList *policyhierarchy_v1.PolicyNodeList) ([]ResourceQuotaAction, error) {

	resourceQuotaList, err := s.client.Kubernetes().CoreV1().ResourceQuotas(meta_v1.NamespaceAll).List(meta_v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", ResourceQuotaObjectName).String(),
	})
	if err != nil {
		return nil, err
	}

	return s.computeResourceQuotaActionsWithResourceQuotaList(resourceQuotaList, policyNodeList), nil
}

func (s *Syncer) computeResourceQuotaActionsWithResourceQuotaList(resourceQuotaList *core_v1.ResourceQuotaList,
	policyNodeList *policyhierarchy_v1.PolicyNodeList) []ResourceQuotaAction {

	existing := map[string]core_v1.ResourceQuota{}
	for _, rq := range resourceQuotaList.Items {
		existing[rq.Namespace] = rq
	}

	declaring := map[string]core_v1.ResourceQuotaSpec{}
	for _, pn := range policyNodeList.Items {
		// TODO(mdruskin): If not working namespace, we should create a hierarchical resource quota
		if pn.Spec.WorkingNamespace {
			declaring[pn.Name] = pn.Spec.Policies.ResourceQuotas[0]
		}
	}

	actions := []ResourceQuotaAction{}
	// Creates and updates
	for ns, rq := range declaring {
		if _, exists := existing[ns]; !exists {
			actions = append(actions, NewResourceQuotaCreateAction(s.client.Kubernetes(), ns, rq))
		} else if !reflect.DeepEqual(rq, existing[ns].Spec) {
			actions = append(actions, NewResourceQuotaUpdateAction(s.client.Kubernetes(), ns, rq, existing[ns].ResourceVersion))
		}
	}
	// Deletions
	for ns, _ := range existing {
		if _, exists := declaring[ns]; !exists {
			actions = append(actions, NewResourceQuotaDeleteAction(s.client.Kubernetes(), ns))
		}
	}

	return actions
}

func (s *Syncer) initialSync() (int64, error) {
	glog.Info("Performing initial sync on namespaces")

	policyNodeList, err := s.client.PolicyHierarchy().K8usV1().PolicyNodes().List(meta_v1.ListOptions{})
	if err != nil {
		return 0, err
	}
	policyNodeResourceVersion, err := strconv.ParseInt(policyNodeList.ResourceVersion, 10, 64)
	if err != nil {
		return 0, err
	}

	glog.Infof("Listed policy nodes at resource version at %d", policyNodeResourceVersion)

	namespaceActions, err := s.computeNamespaceActions(policyNodeList)
	if err != nil {
		return 0, err
	}
	for _, action := range namespaceActions {
		if *dryRun {
			glog.Infof("DryRun: Would execute namespace action %s on namespace %s", action.Operation(), action.Name())
			continue
		}
		err := action.Execute()
		if err != nil {
			glog.Infof("Namespace Action %s %s failed due to %s", action.Name(), action.Operation(), err)
			return 0, err
		}
	}

	resourceQuotaActions, err := s.computeResourceQuotaActions(policyNodeList)
	if err != nil {
		return 0, err
	}
	for _, action := range resourceQuotaActions {
		if *dryRun {
			glog.Infof("DryRun: Would execute resource quota action %s on namespace %s", action.Operation(), action.Name())
			continue
		}
		err := action.Execute()
		if err != nil {
			glog.Infof("Resource Quota Action %s %s failed due to %s", action.Name(), action.Operation(), err)
			return 0, err
		}
	}

	glog.Infof("Finished initial sync, will use resource version %s", policyNodeResourceVersion)
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

func (s *Syncer) onPolicyNodeEvent(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	s.onPolicyNodeEventNamespace(eventType, policyNode)
	s.onPolicyNodeEventResourceQuota(eventType, policyNode)
}

func (s *Syncer) getEventNamespaceAction(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) client.NamespaceAction {
	var action client.NamespaceAction
	namespace := policyNode.Name
	glog.V(3).Infof("Got event %s namespace %s resourceVersion %s", eventType, policyNode.Name, policyNode.ResourceVersion)
	switch eventType {
	case policynodewatcher.Added:
		action = client.NewNamespaceCreateAction(s.client.Kubernetes(), namespace)
	case policynodewatcher.Deleted:
		action = client.NewNamespaceDeleteAction(s.client.Kubernetes(), namespace)
	case policynodewatcher.Modified:
		glog.Infof("Got modified event for %s, ignoring", policyNode.Name)
	}
	return action
}

func (s *Syncer) onPolicyNodeEventNamespace(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	// NOTE: There is a bug here where the namespace create action can operate on a namespace that is presently
	// terminating.  This needs to understand when the namespace is finally deleted then attempt creation, preferably
	// with some sort of timeout.
	action := s.getEventNamespaceAction(eventType, policyNode)
	if action == nil {
		return
	}

	if *dryRun {
		glog.Infof("DryRun: Would execute namespace action %s on namespace %s", action.Operation(), action.Name())
		return
	}
	err := action.Execute()
	if err != nil {
		s.errorCallback(errors.Wrapf(
			err, "Failed to perform namespace action %s on %s: %s", action.Operation(), action.Name(), err))
		return
	}
}

func (s *Syncer) getEventResourceQuotaAction(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) ResourceQuotaAction {
	namespace := policyNode.Name
	glog.V(3).Infof("Got event %s namespace %s resourceVersion %s", eventType, policyNode.Name, policyNode.ResourceVersion)
	switch eventType {
	case policynodewatcher.Added:
		if policyNode.Spec.WorkingNamespace && len(policyNode.Spec.Policies.ResourceQuotas) > 0 {
			return NewResourceQuotaCreateAction(s.client.Kubernetes(), namespace, policyNode.Spec.Policies.ResourceQuotas[0])
		}
	case policynodewatcher.Deleted:
		glog.Infof("Got deleted policy node event %s, ignoring since the resource quota will be auto-deleted", namespace)
	case policynodewatcher.Modified:
		var neededResourceQuotaSpec *core_v1.ResourceQuotaSpec
		if policyNode.Spec.WorkingNamespace && len(policyNode.Spec.Policies.ResourceQuotas) > 0 {
			neededResourceQuotaSpec = &policyNode.Spec.Policies.ResourceQuotas[0]
		}
		// TODO: Replace with with a get from the informer instead.
		existingResourceQuota, _ := s.client.Kubernetes().CoreV1().ResourceQuotas(namespace).Get(ResourceQuotaObjectName, meta_v1.GetOptions{})
		if existingResourceQuota == nil && neededResourceQuotaSpec != nil {
			return NewResourceQuotaCreateAction(s.client.Kubernetes(), namespace, *neededResourceQuotaSpec)
		}
		if existingResourceQuota != nil && neededResourceQuotaSpec == nil {
			return NewResourceQuotaDeleteAction(s.client.Kubernetes(), namespace)
		}
		if existingResourceQuota != nil && neededResourceQuotaSpec != nil && !reflect.DeepEqual(existingResourceQuota.Spec, *neededResourceQuotaSpec) {
			return NewResourceQuotaUpdateAction(s.client.Kubernetes(), namespace, *neededResourceQuotaSpec, existingResourceQuota.ObjectMeta.ResourceVersion)
		}
	}
	return nil
}

func (s *Syncer) onPolicyNodeEventResourceQuota(eventType policynodewatcher.EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	action := s.getEventResourceQuotaAction(eventType, policyNode)
	if action == nil {
		return
	}

	if *dryRun {
		glog.Infof("DryRun: Would execute resource quota action %s on namespace %s", action.Operation(), action.Name())
		return
	}
	err := action.Execute()
	if err != nil {
		s.errorCallback(errors.Wrapf(
			err, "Failed to perform resource quota action %s on %s: %s", action.Operation(), action.Name(), err))
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
