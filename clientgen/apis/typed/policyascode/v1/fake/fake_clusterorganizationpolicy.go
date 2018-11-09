/*
Copyright 2018 The Nomos Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	policyascode_v1 "github.com/google/nomos/pkg/api/policyascode/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterOrganizationPolicies implements ClusterOrganizationPolicyInterface
type FakeClusterOrganizationPolicies struct {
	Fake *FakeBespinV1
}

var clusterorganizationpoliciesResource = schema.GroupVersionResource{Group: "bespin.dev", Version: "v1", Resource: "clusterorganizationpolicies"}

var clusterorganizationpoliciesKind = schema.GroupVersionKind{Group: "bespin.dev", Version: "v1", Kind: "ClusterOrganizationPolicy"}

// Get takes name of the clusterOrganizationPolicy, and returns the corresponding clusterOrganizationPolicy object, and an error if there is any.
func (c *FakeClusterOrganizationPolicies) Get(name string, options v1.GetOptions) (result *policyascode_v1.ClusterOrganizationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusterorganizationpoliciesResource, name), &policyascode_v1.ClusterOrganizationPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.ClusterOrganizationPolicy), err
}

// List takes label and field selectors, and returns the list of ClusterOrganizationPolicies that match those selectors.
func (c *FakeClusterOrganizationPolicies) List(opts v1.ListOptions) (result *policyascode_v1.ClusterOrganizationPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusterorganizationpoliciesResource, clusterorganizationpoliciesKind, opts), &policyascode_v1.ClusterOrganizationPolicyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyascode_v1.ClusterOrganizationPolicyList{ListMeta: obj.(*policyascode_v1.ClusterOrganizationPolicyList).ListMeta}
	for _, item := range obj.(*policyascode_v1.ClusterOrganizationPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterOrganizationPolicies.
func (c *FakeClusterOrganizationPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusterorganizationpoliciesResource, opts))
}

// Create takes the representation of a clusterOrganizationPolicy and creates it.  Returns the server's representation of the clusterOrganizationPolicy, and an error, if there is any.
func (c *FakeClusterOrganizationPolicies) Create(clusterOrganizationPolicy *policyascode_v1.ClusterOrganizationPolicy) (result *policyascode_v1.ClusterOrganizationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusterorganizationpoliciesResource, clusterOrganizationPolicy), &policyascode_v1.ClusterOrganizationPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.ClusterOrganizationPolicy), err
}

// Update takes the representation of a clusterOrganizationPolicy and updates it. Returns the server's representation of the clusterOrganizationPolicy, and an error, if there is any.
func (c *FakeClusterOrganizationPolicies) Update(clusterOrganizationPolicy *policyascode_v1.ClusterOrganizationPolicy) (result *policyascode_v1.ClusterOrganizationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusterorganizationpoliciesResource, clusterOrganizationPolicy), &policyascode_v1.ClusterOrganizationPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.ClusterOrganizationPolicy), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusterOrganizationPolicies) UpdateStatus(clusterOrganizationPolicy *policyascode_v1.ClusterOrganizationPolicy) (*policyascode_v1.ClusterOrganizationPolicy, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(clusterorganizationpoliciesResource, "status", clusterOrganizationPolicy), &policyascode_v1.ClusterOrganizationPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.ClusterOrganizationPolicy), err
}

// Delete takes name of the clusterOrganizationPolicy and deletes it. Returns an error if one occurs.
func (c *FakeClusterOrganizationPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clusterorganizationpoliciesResource, name), &policyascode_v1.ClusterOrganizationPolicy{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterOrganizationPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusterorganizationpoliciesResource, listOptions)

	_, err := c.Fake.Invokes(action, &policyascode_v1.ClusterOrganizationPolicyList{})
	return err
}

// Patch applies the patch and returns the patched clusterOrganizationPolicy.
func (c *FakeClusterOrganizationPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyascode_v1.ClusterOrganizationPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterorganizationpoliciesResource, name, data, subresources...), &policyascode_v1.ClusterOrganizationPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.ClusterOrganizationPolicy), err
}
