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
	typedv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
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
	client typedv1.NomosV1Interface, pnLister listersv1.PolicyNodeLister, cpLister listersv1.ClusterPolicyLister) Factories {
	return Factories{newPolicyNodeActionFactory(client, pnLister), newClusterPolicyActionFactory(client, cpLister)}
}

type policyNodeActionFactory struct {
	*action.ReflectiveActionSpec
}

func newPolicyNodeActionFactory(client typedv1.NomosV1Interface, lister listersv1.PolicyNodeLister) policyNodeActionFactory {
	return policyNodeActionFactory{policynode.NewActionSpec(client, lister)}
}

// NewCreate returns an action for creating PolicyNodes.
func (f policyNodeActionFactory) NewCreate(policyNode *policyhierarchyv1.PolicyNode) action.Interface {
	return action.NewReflectiveCreateAction("", policyNode.Name, policyNode, f.ReflectiveActionSpec)
}

// NewUpdate returns an action for updating PolicyNodes. This action ignores the ResourceVersion of
// the new PolicyNode as well as most of the Status. If Status.SyncState has been set then that will
// be copied over.
func (f policyNodeActionFactory) NewUpdate(policyNode *policyhierarchyv1.PolicyNode) action.Interface {
	updatePolicy := func(old runtime.Object) (runtime.Object, error) {
		newPN := policyNode.DeepCopy()
		oldPN := old.(*policyhierarchyv1.PolicyNode)
		newPN.ResourceVersion = oldPN.ResourceVersion
		newSyncState := newPN.Status.SyncState
		oldPN.Status.DeepCopyInto(&newPN.Status)
		if !newSyncState.IsUnknown() {
			newPN.Status.SyncState = newSyncState
		}
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
	client typedv1.NomosV1Interface,
	lister listersv1.ClusterPolicyLister) clusterPolicyActionFactory {
	return clusterPolicyActionFactory{clusterpolicy.NewActionSpec(client, lister)}
}

// NewCreate returns an action for creating ClusterPolicies.
func (f clusterPolicyActionFactory) NewCreate(clusterPolicy *policyhierarchyv1.ClusterPolicy) action.Interface {
	return action.NewReflectiveCreateAction("", clusterPolicy.Name, clusterPolicy, f.ReflectiveActionSpec)
}

// NewUpdate returns an action for updating ClusterPolicies. This action ignores the ResourceVersion
// of the new ClusterPolicy as well as most of the Status. If Status.SyncState has been set then
// that will be copied over.
func (f clusterPolicyActionFactory) NewUpdate(clusterPolicy *policyhierarchyv1.ClusterPolicy) action.Interface {
	updatePolicy := func(old runtime.Object) (runtime.Object, error) {
		newCP := clusterPolicy.DeepCopy()
		oldCP := old.(*policyhierarchyv1.ClusterPolicy)
		newCP.ResourceVersion = oldCP.ResourceVersion
		newSyncState := newCP.Status.SyncState
		oldCP.Status.DeepCopyInto(&newCP.Status)
		if !newSyncState.IsUnknown() {
			newCP.Status.SyncState = newSyncState
		}
		return newCP, nil
	}
	return action.NewReflectiveUpdateAction("", clusterPolicy.Name, updatePolicy, f.ReflectiveActionSpec)
}

// NewDelete returns an action for deleting ClusterPolicies.
func (f clusterPolicyActionFactory) NewDelete(
	clusterPolicyName string) action.Interface {
	return action.NewReflectiveDeleteAction("", clusterPolicyName, f.ReflectiveActionSpec)
}
