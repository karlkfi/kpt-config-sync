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

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/action"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewPolicyNodeUpsertAction creates an action for upserting PolicyNodes.
func NewPolicyNodeUpsertAction(
	policyNode *policyhierarchy_v1.PolicyNode,
	spec *action.ReflectiveActionSpec) action.Interface {
	return action.NewReflectiveUpsertAction("", policyNode.Name, policyNode, spec)
}

// NewPolicyNodeDeleteAction creates an action for deleting PolicyNodes.
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
	client typed_v1.StolosV1Interface,
	lister listers_v1.PolicyNodeLister) *action.ReflectiveActionSpec {
	return &action.ReflectiveActionSpec{
		KindPlural: "PolicyNodes",
		EqualSpec:  PolicyNodesEqual,
		Client:     client,
		Lister:     lister,
	}
}
