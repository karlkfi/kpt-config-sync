// Reviewed by sunilarora
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

package actions

import (
	"reflect"

	api_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	listers_v1 "github.com/google/nomos/pkg/client/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/nomos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewPolicyNodeUpsertAction nodeCreates an action for upserting PolicyNodes.
func NewPolicyNodeUpsertAction(
	policyNode *policyhierarchy_v1.PolicyNode,
	spec *action.ReflectiveActionSpec) action.Interface {
	return action.NewReflectiveUpsertAction("", policyNode.Name, policyNode, spec)
}

// NewPolicyNodeDeleteAction nodeCreates an action for deleting PolicyNodes.
func NewPolicyNodeDeleteAction(
	policyNode *policyhierarchy_v1.PolicyNode,
	spec *action.ReflectiveActionSpec) action.Interface {
	return action.NewReflectiveDeleteAction("", policyNode.Name, spec)
}

func PolicyNodesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRole := lhs.(*policyhierarchy_v1.PolicyNode)
	rRole := rhs.(*policyhierarchy_v1.PolicyNode)
	return reflect.DeepEqual(lRole.Spec, rRole.Spec)
}

func NewPolicyNodeActionSpec(
	client typed_v1.NomosV1Interface,
	lister listers_v1.PolicyNodeLister) *action.ReflectiveActionSpec {

	return &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(policyhierarchy_v1.PolicyNode{}),
		KindPlural: action.Plural(policyhierarchy_v1.PolicyNode{}),
		Group:      api_v1.GroupName,
		Version:    api_v1.SchemeGroupVersion.Version,
		EqualSpec:  PolicyNodesEqual,
		Client:     client,
		Lister:     lister,
	}
}

// NewClusterPolicyUpsertAction nodeCreates an action for upserting ClusterPolicies.
func NewClusterPolicyUpsertAction(
	clusterPolicy *policyhierarchy_v1.ClusterPolicy,
	spec *action.ReflectiveActionSpec) action.Interface {
	return action.NewReflectiveUpsertAction("", clusterPolicy.Name, clusterPolicy, spec)
}

// NewClusterPolicyDeleteAction nodeCreates an action for deleting ClusterPolicies.
func NewClusterPolicyDeleteAction(
	clusterPolicy *policyhierarchy_v1.ClusterPolicy,
	spec *action.ReflectiveActionSpec) action.Interface {
	return action.NewReflectiveDeleteAction("", clusterPolicy.Name, spec)
}

func ClusterPoliciesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRole := lhs.(*policyhierarchy_v1.ClusterPolicy)
	rRole := rhs.(*policyhierarchy_v1.ClusterPolicy)
	return reflect.DeepEqual(lRole.Spec, rRole.Spec)
}

func NewClusterPolicyActionSpec(
	client typed_v1.NomosV1Interface,
	lister listers_v1.ClusterPolicyLister) *action.ReflectiveActionSpec {
	return &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(policyhierarchy_v1.ClusterPolicy{}),
		KindPlural: action.Plural(policyhierarchy_v1.ClusterPolicy{}),
		Group:      api_v1.GroupName,
		Version:    api_v1.SchemeGroupVersion.Version,
		EqualSpec:  ClusterPoliciesEqual,
		Client:     client,
		Lister:     lister,
	}
}
