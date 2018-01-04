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
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePolicyNodes implements PolicyNodeInterface
type FakePolicyNodes struct {
	Fake *FakeK8usV1
}

var policynodesResource = schema.GroupVersionResource{Group: "k8us.k8s.io", Version: "v1", Resource: "policynodes"}

var policynodesKind = schema.GroupVersionKind{Group: "k8us.k8s.io", Version: "v1", Kind: "PolicyNode"}

// Get takes name of the policyNode, and returns the corresponding policyNode object, and an error if there is any.
func (c *FakePolicyNodes) Get(name string, options v1.GetOptions) (result *policyhierarchy_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(policynodesResource, name), &policyhierarchy_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.PolicyNode), err
}

// List takes label and field selectors, and returns the list of PolicyNodes that match those selectors.
func (c *FakePolicyNodes) List(opts v1.ListOptions) (result *policyhierarchy_v1.PolicyNodeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(policynodesResource, policynodesKind, opts), &policyhierarchy_v1.PolicyNodeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyhierarchy_v1.PolicyNodeList{}
	for _, item := range obj.(*policyhierarchy_v1.PolicyNodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested policyNodes.
func (c *FakePolicyNodes) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(policynodesResource, opts))
}

// Create takes the representation of a policyNode and creates it.  Returns the server's representation of the policyNode, and an error, if there is any.
func (c *FakePolicyNodes) Create(policyNode *policyhierarchy_v1.PolicyNode) (result *policyhierarchy_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(policynodesResource, policyNode), &policyhierarchy_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.PolicyNode), err
}

// Update takes the representation of a policyNode and updates it. Returns the server's representation of the policyNode, and an error, if there is any.
func (c *FakePolicyNodes) Update(policyNode *policyhierarchy_v1.PolicyNode) (result *policyhierarchy_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(policynodesResource, policyNode), &policyhierarchy_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.PolicyNode), err
}

// Delete takes name of the policyNode and deletes it. Returns an error if one occurs.
func (c *FakePolicyNodes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(policynodesResource, name), &policyhierarchy_v1.PolicyNode{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePolicyNodes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(policynodesResource, listOptions)

	_, err := c.Fake.Invokes(action, &policyhierarchy_v1.PolicyNodeList{})
	return err
}

// Patch applies the patch and returns the patched policyNode.
func (c *FakePolicyNodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyhierarchy_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(policynodesResource, name, data, subresources...), &policyhierarchy_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.PolicyNode), err
}
