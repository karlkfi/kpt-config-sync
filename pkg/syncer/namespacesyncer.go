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
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/util/namespaceutil"
	"github.com/google/stolos/pkg/util/set/stringset"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
	"github.com/google/stolos/pkg/syncer/actions"
)

// NamespaceSyncer handles syncing namespaces from policy nodes.
type NamespaceSyncer struct {
	client          meta.Interface
	namespaceLister listers_core_v1.NamespaceLister
}

var _ PolicyNodeSyncerInterface = &NamespaceSyncer{}

// NewNamespaceSyncer creates a new namespace syncer from the client.
func NewNamespaceSyncer(
	client meta.Interface,
	namespaceLister listers_core_v1.NamespaceLister) *NamespaceSyncer {
	return &NamespaceSyncer{
		client:          client,
		namespaceLister: namespaceLister,
	}
}

// InitialSync implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) InitialSync(nodes []*policyhierarchy_v1.PolicyNode) error {
	existingNamespaces, err := s.namespaceLister.List(labels.Everything())
	if err != nil {
		return err
	}

	namespaceActions := s.computeActions(existingNamespaces, nodes)
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
	return s.runAction(actions.NewNamespaceCreateAction(s.client.Kubernetes(), node.Name))
}

// OnUpdate implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnUpdate(
	old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	// Noop
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (s *NamespaceSyncer) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	return s.runAction(actions.NewNamespaceDeleteAction(s.client.Kubernetes(), node.Name))
}

func (s *NamespaceSyncer) runAction(action actions.NamespaceAction) error {
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
	existingNamespaceList []*core_v1.Namespace,
	nodes []*policyhierarchy_v1.PolicyNode) []actions.NamespaceAction {
	existingNamespaces := stringset.New()
	for _, namespace := range existingNamespaceList {
		if namespaceutil.IsReserved(*namespace) {
			continue
		}
		switch namespace.Status.Phase {
		case core_v1.NamespaceActive:
			existingNamespaces.Add(namespace.Name)
		case core_v1.NamespaceTerminating:
			// noop, handled by namespaceactions
		}
	}

	declaredNamespaces := stringset.New()
	for _, policyNode := range nodes {
		declaredNamespaces.Add(policyNode.ObjectMeta.Name)
	}

	needsCreate := declaredNamespaces.Difference(existingNamespaces)
	needsDelete := existingNamespaces.Difference(declaredNamespaces)

	namespaceActions := []actions.NamespaceAction{}
	needsCreate.ForEach(func(ns string) {
		glog.Infof("Adding create operation for %s", ns)
		namespaceActions = append(namespaceActions, actions.NewNamespaceCreateAction(s.client.Kubernetes(), ns))
	})
	needsDelete.ForEach(func(ns string) {
		glog.Infof("Adding delete operation for %s", ns)
		namespaceActions = append(namespaceActions, actions.NewNamespaceDeleteAction(s.client.Kubernetes(), ns))
	})

	return namespaceActions
}
