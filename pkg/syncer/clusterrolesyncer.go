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
	"reflect"

	"github.com/google/stolos/pkg/syncer/labeling"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/action"
	"github.com/google/stolos/pkg/client/meta"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	listers_rbac_v1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/util/workqueue"
)

// ClusterRoleUnpacker is an unpacker for the ClusterRole resource
type ClusterRoleUnpacker struct {
	// client *meta.Client
	lister listers_rbac_v1.ClusterRoleLister
	spec   *action.ReflectiveActionSpec
}

// ClusterRoleUnpacker implements ClusterPolicyUnpackerInterface
var _ ClusterPolicyUnpackerInterface = &ClusterRoleUnpacker{}

// ClusterRoleEquals compares two ClusterRole objects for equivalence.  AggregationRule defines a
// way to generate the role from other roles, so if one side of the comparison is non-nil, then we
// compare on aggregation rules, otherwise we compare on Rules.
func ClusterRoleEquals(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	lhs := lhsObj.(*rbac_v1.ClusterRole)
	rhs := rhsObj.(*rbac_v1.ClusterRole)

	if lhs.AggregationRule != nil || rhs.AggregationRule != nil {
		return reflect.DeepEqual(lhs.AggregationRule, rhs.AggregationRule)
	}
	return reflect.DeepEqual(lhs.Rules, rhs.Rules)
}

// NewClusterRoleUnpacker creates a new object for handling syncing cluster roles.
func NewClusterRoleUnpacker(
	client meta.Interface, lister listers_rbac_v1.ClusterRoleLister) *ClusterRoleUnpacker {
	return &ClusterRoleUnpacker{
		lister: lister,
		spec: &action.ReflectiveActionSpec{
			KindPlural: action.Plural(reflect.TypeOf(rbac_v1.ClusterRole{}).Name()),
			EqualSpec:  func(runtime.Object, runtime.Object) bool { return false },
			Client:     client.Kubernetes().RbacV1(),
			Lister:     lister,
		},
	}
}

// UpdateRemovals implements UnpackerInterface
func (s *ClusterRoleUnpacker) UpdateRemovals(
	oldNode *policyhierarchy_v1.ClusterPolicy,
	newNode *policyhierarchy_v1.ClusterPolicy) []runtime.Object {
	ret := []runtime.Object{}
	decl := map[string]bool{}
	for _, clusterRole := range newNode.Spec.Policies.ClusterRolesV1 {
		decl[clusterRole.Name] = true
	}
	for _, clusterRole := range oldNode.Spec.Policies.ClusterRolesV1 {
		if !decl[clusterRole.Name] {
			ret = append(ret, clusterRole.DeepCopy())
		}
	}
	return ret
}

// Upserts implements UnpackerInterface
func (s *ClusterRoleUnpacker) Upserts(node *policyhierarchy_v1.ClusterPolicy) []runtime.Object {
	ret := []runtime.Object{}
	for _, clusterRole := range node.Spec.Policies.ClusterRolesV1 {
		newClusterRole := clusterRole.DeepCopy()
		blockOwnerDeletion := true
		newClusterRole.ObjectMeta.OwnerReferences = []meta_v1.OwnerReference{
			meta_v1.OwnerReference{
				APIVersion:         policyhierarchy_v1.SchemeGroupVersion.String(),
				Kind:               "ClusterPolicy",
				Name:               node.Name,
				UID:                node.UID,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		}
		labeling.AddOriginLabel(&newClusterRole.ObjectMeta)
		ret = append(ret, newClusterRole)
	}
	return ret
}

// Names implements UnpackerInterface
func (s *ClusterRoleUnpacker) Names(node *policyhierarchy_v1.ClusterPolicy) map[string]bool {
	ret := map[string]bool{}
	for _, clusterRole := range node.Spec.Policies.ClusterRolesV1 {
		ret[clusterRole.Name] = true
	}
	return ret
}

// List will list the resource and return all items as a runtime.Object
func (s *ClusterRoleUnpacker) List() ([]runtime.Object, error) {
	clusterRoles, err := s.lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var ret []runtime.Object
	for _, clusterRole := range clusterRoles {
		ret = append(ret, clusterRole)
	}
	return ret, nil
}

func (s *ClusterRoleUnpacker) NewUpsertAction(name string, obj runtime.Object) action.Interface {
	return action.NewReflectiveUpsertAction("", name, obj, s.spec)
}

func (s *ClusterRoleUnpacker) NewDeleteAction(name string) action.Interface {
	return action.NewReflectiveDeleteAction("", name, s.spec)
}

// NewClusterRoleSyncer creates the RBAC syncer object
func NewClusterRoleSyncer(
	client meta.Interface,
	lister listers_rbac_v1.ClusterRoleLister,
	queue workqueue.RateLimitingInterface) *ClusterGenericSyncer {
	return NewClusterGenericSyncer(NewClusterRoleUnpacker(client, lister), queue)
}
