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
	listers_v1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/nomos/clientgen/policyhierarchy/typed/policyhierarchy/v1"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/policynode"
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
	return policyNodeActionFactory{policynode.NewActionSpec(client, lister)}
}

// NewCreate returns an action for creating PolicyNodes.
func (f policyNodeActionFactory) NewCreate(policyNode *policyhierarchy_v1.PolicyNode) action.Interface {
	return action.NewReflectiveCreateAction("", policyNode.Name, policyNode, f.ReflectiveActionSpec)
}

// NewUpdate returns an action for updating PolicyNodes. This action ignores the Status and
// ResourceVersion of the new PolicyNode.
func (f policyNodeActionFactory) NewUpdate(policyNode *policyhierarchy_v1.PolicyNode) action.Interface {
	updatePolicy := func(old runtime.Object) (runtime.Object, error) {
		newPN := policyNode.DeepCopy()
		oldPN := old.(*policyhierarchy_v1.PolicyNode)
		newPN.ResourceVersion = oldPN.ResourceVersion
		oldPN.Status.DeepCopyInto(&newPN.Status)
		return newPN, nil
	}
	return action.NewReflectiveUpdateAction("", policyNode.Name, updatePolicy, f.ReflectiveActionSpec)
}

// NewDelete returns an action for deleting PolicyNodes.
func (f policyNodeActionFactory) NewDelete(nodeName string) action.Interface {
	return action.NewReflectiveDeleteAction("", nodeName, f.ReflectiveActionSpec)
}

type clusterPolicyActionFactory struct {
	*action.ReflectiveActionSpec
}

func newClusterPolicyActionFactory(
	client typed_v1.NomosV1Interface,
	lister listers_v1.ClusterPolicyLister) clusterPolicyActionFactory {
	return clusterPolicyActionFactory{clusterpolicy.NewActionSpec(client, lister)}
}

// NewCreate returns an action for creating ClusterPolicies.
func (f clusterPolicyActionFactory) NewCreate(clusterPolicy *policyhierarchy_v1.ClusterPolicy) action.Interface {
	return action.NewReflectiveCreateAction("", clusterPolicy.Name, clusterPolicy, f.ReflectiveActionSpec)
}

// NewUpdate returns an action for updating ClusterPolicies. This action ignores the Status and
// ResourceVersion of the new ClusterPolicy.
func (f clusterPolicyActionFactory) NewUpdate(clusterPolicy *policyhierarchy_v1.ClusterPolicy) action.Interface {
	updatePolicy := func(old runtime.Object) (runtime.Object, error) {
		newCP := clusterPolicy.DeepCopy()
		oldCP := old.(*policyhierarchy_v1.ClusterPolicy)
		newCP.ResourceVersion = oldCP.ResourceVersion
		oldCP.Status.DeepCopyInto(&newCP.Status)
		return newCP, nil
	}
	return action.NewReflectiveUpdateAction("", clusterPolicy.Name, updatePolicy, f.ReflectiveActionSpec)
}

// NewDelete returns an action for deleting ClusterPolicies.
func (f clusterPolicyActionFactory) NewDelete(
	clusterPolicyName string) action.Interface {
	return action.NewReflectiveDeleteAction("", clusterPolicyName, f.ReflectiveActionSpec)
}
