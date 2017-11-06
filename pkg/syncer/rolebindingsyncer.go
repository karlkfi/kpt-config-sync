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
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/syncer/actions"
	listers_rbac_v1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/util/workqueue"
)

// RoleBindingUnpacker is an unpacker for the RoleBinding resource
type RoleBindingUnpacker struct {
}

// RoleBindingUnpacker implements UnpackerInterface
var _ UnpackerInterface = &RoleBindingUnpacker{}

// NewRoleBindingUnpacker creates a new object for handling syncing role bindings.
func NewRoleBindingUnpacker() *RoleBindingUnpacker {
	return &RoleBindingUnpacker{}
}

// UpdateRemovals implements UnpackerInterface
func (s *RoleBindingUnpacker) UpdateRemovals(
	oldNode *policyhierarchy_v1.PolicyNode,
	newNode *policyhierarchy_v1.PolicyNode) []interface{} {
	ret := []interface{}{}
	decl := map[string]bool{}
	for _, roleBinding := range newNode.Spec.Policies.RoleBindings {
		decl[roleBinding.Name] = true
	}
	for _, roleBinding := range oldNode.Spec.Policies.RoleBindings {
		if !decl[roleBinding.Name] {
			newRoleBinding := roleBinding.DeepCopy()
			newRoleBinding.Namespace = oldNode.Name
			ret = append(ret, newRoleBinding)
		}
	}
	return ret
}

// Upserts implements UnpackerInterface
func (s *RoleBindingUnpacker) Upserts(node *policyhierarchy_v1.PolicyNode) []interface{} {
	ret := []interface{}{}
	for _, roleBinding := range node.Spec.Policies.RoleBindings {
		newRoleBinding := roleBinding.DeepCopy()
		newRoleBinding.Namespace = node.Name
		ret = append(ret, newRoleBinding)
	}
	return ret
}

// Names implements UnpackerInterface
func (s *RoleBindingUnpacker) Names(node *policyhierarchy_v1.PolicyNode) map[string]bool {
	ret := map[string]bool{}
	for _, roleBinding := range node.Spec.Policies.RoleBindings {
		ret[roleBinding.Name] = true
	}
	return ret
}

// NewRoleBindingSyncer creates the RBAC syncer object
func NewRoleBindingSyncer(
	client meta.Interface,
	lister listers_rbac_v1.RoleBindingLister,
	queue workqueue.RateLimitingInterface) *GenericSyncer {
	return NewGenericSyncer(
		actions.NewRoleBindingResource(client.Kubernetes(), lister), NewRoleBindingUnpacker(), queue)
}
