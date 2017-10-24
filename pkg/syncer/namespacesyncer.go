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
	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/util/namespaceutil"
	"github.com/google/stolos/pkg/util/set/stringset"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceSyncer handles syncing namespaces from policy nodes.
type NamespaceSyncer struct {
	client meta.Interface
}

var _ PolicyNodeSyncerInterface = &NamespaceSyncer{}

func NewNamespaceSyncer(client meta.Interface) *NamespaceSyncer {
	return &NamespaceSyncer{
		client: client,
	}
}

// InitialSync implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) InitialSync(nodes []*policyhierarchy_v1.PolicyNode) error {
	// TODO: Use informer for this list operation
	existingNamespaceList, err := s.client.Kubernetes().CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	namespaceActions := s.computeActions(existingNamespaceList, nodes)
	for _, action := range namespaceActions {
		if *dryRun {
			glog.Infof("DryRun: Would execute namespace action %s on namespace %s", action.Operation(), action.Name())
			continue
		}
		err := action.Execute()
		if err != nil {
			glog.Infof("Namespace Action %s %s failed due to %s", action.Name(), action.Operation(), err)
			return err
		}
	}
	return nil
}

// OnCreate implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnCreate(node *policyhierarchy_v1.PolicyNode) error {
	return s.runAction(client.NewNamespaceCreateAction(s.client.Kubernetes(), node.Name))
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnUpdate(
	old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	// Noop
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	return s.runAction(client.NewNamespaceDeleteAction(s.client.Kubernetes(), node.Name))
}

func (s *NamespaceSyncer) runAction(action client.NamespaceAction) error {
	if *dryRun {
		glog.Infof("DryRun: Would execute namespace action %s on namespace %s", action.Operation(), action.Name())
		return nil
	}
	err := action.Execute()
	if err != nil {
		return errors.Wrapf(
			err, "Failed to perform namespace action %s on %s: %s", action.Operation(), action.Name(), err)
	}
	return nil
}

// computeActions determines which namespaces to create and delete on initial sync.
func (s *NamespaceSyncer) computeActions(
	existingNamespaceList *core_v1.NamespaceList,
	nodes []*policyhierarchy_v1.PolicyNode) []client.NamespaceAction {
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
	for _, policyNode := range nodes {
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
