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

package syncer

import (
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/syncer/actions"
	listers_rbac_v1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/util/workqueue"
)

// RoleUnpacker is an unpacker for the Role resource
type RoleUnpacker struct {
}

// RoleUnpacker implements UnpackerInterface
var _ UnpackerInterface = &RoleUnpacker{}

// NewRoleUnpacker creates a new object for handling syncing role bindings.
func NewRoleUnpacker() *RoleUnpacker {
	return &RoleUnpacker{}
}

// UpdateRemovals implements UnpackerInterface
func (s *RoleUnpacker) UpdateRemovals(
	oldNode *policyhierarchy_v1.PolicyNode,
	newNode *policyhierarchy_v1.PolicyNode) []interface{} {
	ret := []interface{}{}
	decl := map[string]bool{}
	for _, role := range newNode.Spec.Policies.Roles {
		decl[role.Name] = true
	}
	for _, role := range oldNode.Spec.Policies.Roles {
		if !decl[role.Name] {
			newRole := role.DeepCopy()
			newRole.Namespace = oldNode.Name
			ret = append(ret, newRole)
		}
	}
	return ret
}

// Upserts implements UnpackerInterface
func (s *RoleUnpacker) Upserts(node *policyhierarchy_v1.PolicyNode) []interface{} {
	ret := []interface{}{}
	for _, role := range node.Spec.Policies.Roles {
		newRole := role.DeepCopy()
		newRole.Namespace = node.Name
		ret = append(ret, newRole)
	}
	return ret
}

// Names implements UnpackerInterface
func (s *RoleUnpacker) Names(node *policyhierarchy_v1.PolicyNode) map[string]bool {
	ret := map[string]bool{}
	for _, role := range node.Spec.Policies.Roles {
		ret[role.Name] = true
	}
	return ret
}

// NewRoleSyncer creates the RBAC syncer object
func NewRoleSyncer(
	client meta.Interface,
	lister listers_rbac_v1.RoleLister,
	queue workqueue.RateLimitingInterface) *GenericSyncer {
	return NewGenericSyncer(
		actions.NewRoleResource(client.Kubernetes(), lister), NewRoleUnpacker(), queue)
}
