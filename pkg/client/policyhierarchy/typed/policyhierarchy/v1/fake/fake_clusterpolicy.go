/*
Copyright 2018 The Stolos Authors.

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

package fake

import (
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterPolicies implements ClusterPolicyInterface
type FakeClusterPolicies struct {
	Fake *FakeStolosV1
}

var clusterpoliciesResource = schema.GroupVersionResource{Group: "stolos.dev", Version: "v1", Resource: "clusterpolicies"}

var clusterpoliciesKind = schema.GroupVersionKind{Group: "stolos.dev", Version: "v1", Kind: "ClusterPolicy"}

// Get takes name of the clusterPolicy, and returns the corresponding clusterPolicy object, and an error if there is any.
func (c *FakeClusterPolicies) Get(name string, options v1.GetOptions) (result *policyhierarchy_v1.ClusterPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusterpoliciesResource, name), &policyhierarchy_v1.ClusterPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.ClusterPolicy), err
}

// List takes label and field selectors, and returns the list of ClusterPolicies that match those selectors.
func (c *FakeClusterPolicies) List(opts v1.ListOptions) (result *policyhierarchy_v1.ClusterPolicyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusterpoliciesResource, clusterpoliciesKind, opts), &policyhierarchy_v1.ClusterPolicyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyhierarchy_v1.ClusterPolicyList{}
	for _, item := range obj.(*policyhierarchy_v1.ClusterPolicyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterPolicies.
func (c *FakeClusterPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusterpoliciesResource, opts))
}

// Create takes the representation of a clusterPolicy and creates it.  Returns the server's representation of the clusterPolicy, and an error, if there is any.
func (c *FakeClusterPolicies) Create(clusterPolicy *policyhierarchy_v1.ClusterPolicy) (result *policyhierarchy_v1.ClusterPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusterpoliciesResource, clusterPolicy), &policyhierarchy_v1.ClusterPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.ClusterPolicy), err
}

// Update takes the representation of a clusterPolicy and updates it. Returns the server's representation of the clusterPolicy, and an error, if there is any.
func (c *FakeClusterPolicies) Update(clusterPolicy *policyhierarchy_v1.ClusterPolicy) (result *policyhierarchy_v1.ClusterPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusterpoliciesResource, clusterPolicy), &policyhierarchy_v1.ClusterPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.ClusterPolicy), err
}

// Delete takes name of the clusterPolicy and deletes it. Returns an error if one occurs.
func (c *FakeClusterPolicies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clusterpoliciesResource, name), &policyhierarchy_v1.ClusterPolicy{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusterpoliciesResource, listOptions)

	_, err := c.Fake.Invokes(action, &policyhierarchy_v1.ClusterPolicyList{})
	return err
}

// Patch applies the patch and returns the patched clusterPolicy.
func (c *FakeClusterPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyhierarchy_v1.ClusterPolicy, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterpoliciesResource, name, data, subresources...), &policyhierarchy_v1.ClusterPolicy{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.ClusterPolicy), err
}
