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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/nomos/clientgen/policyhierarchy/typed/policyhierarchy/v1"
	api_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// Factories contains factories for creating actions on Nomos custom resources.
type Factories struct {
	PolicyNodeAction    policyNodeActionFactory
	ClusterPolicyAction clusterPolicyActionFactory
}

// NewFactories creates a new Factories.
func NewFactories(
	client typed_v1.NomosV1Interface, pnLister listers_v1.PolicyNodeLister, cpLister listers_v1.ClusterPolicyLister) Factories {
	return Factories{newPolicyNodeActionFactory(client, pnLister), newClusterPolicyActionFactory(client, cpLister)}
}

type policyNodeActionFactory struct {
	*action.ReflectiveActionSpec
}

func newPolicyNodeActionFactory(client typed_v1.NomosV1Interface, lister listers_v1.PolicyNodeLister) policyNodeActionFactory {
	return policyNodeActionFactory{
		&action.ReflectiveActionSpec{
			Resource:   action.LowerPlural(policyhierarchy_v1.PolicyNode{}),
			KindPlural: action.Plural(policyhierarchy_v1.PolicyNode{}),
			Group:      api_v1.GroupName,
			Version:    api_v1.SchemeGroupVersion.Version,
			EqualSpec:  policyNodesEqual,
			Client:     client,
			Lister:     lister,
		},
	}
}

// NewUpsert nodeCreates an action for upserting PolicyNodes.
func (f policyNodeActionFactory) NewUpsert(policyNode *policyhierarchy_v1.PolicyNode) action.Interface {
	return action.NewReflectiveUpsertAction("", policyNode.Name, policyNode, f.ReflectiveActionSpec)
}

// NewDelete nodeCreates an action for deleting PolicyNodes.
func (f policyNodeActionFactory) NewDelete(nodeName string) action.Interface {
	return action.NewReflectiveDeleteAction("", nodeName, f.ReflectiveActionSpec)
}

var pnsIgnore = []cmp.Option{
	// Quantity has a few unexported fields which we need to explicitly ignore. The path is:
	// PolicyNodeSpec -> ResourceQuota -> ResourceQuotaSpec -> ResourceList -> Quantity
	cmpopts.IgnoreUnexported(resource.Quantity{}),
	cmpopts.IgnoreFields(policyhierarchy_v1.PolicyNodeSpec{}, "ImportToken", "ImportTime"),
}

func policyNodesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*policyhierarchy_v1.PolicyNode)
	r := rhs.(*policyhierarchy_v1.PolicyNode)
	return cmp.Equal(l.Spec, r.Spec, pnsIgnore...)
}

type clusterPolicyActionFactory struct {
	*action.ReflectiveActionSpec
}

func newClusterPolicyActionFactory(
	client typed_v1.NomosV1Interface,
	lister listers_v1.ClusterPolicyLister) clusterPolicyActionFactory {
	return clusterPolicyActionFactory{&action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(policyhierarchy_v1.ClusterPolicy{}),
		KindPlural: action.Plural(policyhierarchy_v1.ClusterPolicy{}),
		Group:      api_v1.GroupName,
		Version:    api_v1.SchemeGroupVersion.Version,
		EqualSpec:  clusterPoliciesEqual,
		Client:     client,
		Lister:     lister,
	},
	}
}

// NewUpsert creates an action for upserting ClusterPolicies.
func (f clusterPolicyActionFactory) NewUpsert(
	clusterPolicy *policyhierarchy_v1.ClusterPolicy) action.Interface {
	return action.NewReflectiveUpsertAction("", clusterPolicy.Name, clusterPolicy, f.ReflectiveActionSpec)
}

// NewDelete creates an action for upserting ClusterPolicies.
func (f clusterPolicyActionFactory) NewDelete(
	clusterPolicyName string) action.Interface {
	return action.NewReflectiveDeleteAction("", clusterPolicyName, f.ReflectiveActionSpec)

}

var cpsIgnore = []cmp.Option{
	cmpopts.IgnoreFields(policyhierarchy_v1.ClusterPolicySpec{}, "ImportToken", "ImportTime"),
}

func clusterPoliciesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*policyhierarchy_v1.ClusterPolicy)
	r := rhs.(*policyhierarchy_v1.ClusterPolicy)
	return cmp.Equal(l.Spec, r.Spec, cpsIgnore...)
}
