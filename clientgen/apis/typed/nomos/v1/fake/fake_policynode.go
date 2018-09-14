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

package fake

import (
	nomos_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePolicyNodes implements PolicyNodeInterface
type FakePolicyNodes struct {
	Fake *FakeNomosV1
}

var policynodesResource = schema.GroupVersionResource{Group: "nomos.dev", Version: "v1", Resource: "policynodes"}

var policynodesKind = schema.GroupVersionKind{Group: "nomos.dev", Version: "v1", Kind: "PolicyNode"}

// Get takes name of the policyNode, and returns the corresponding policyNode object, and an error if there is any.
func (c *FakePolicyNodes) Get(name string, options v1.GetOptions) (result *nomos_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(policynodesResource, name), &nomos_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*nomos_v1.PolicyNode), err
}

// List takes label and field selectors, and returns the list of PolicyNodes that match those selectors.
func (c *FakePolicyNodes) List(opts v1.ListOptions) (result *nomos_v1.PolicyNodeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(policynodesResource, policynodesKind, opts), &nomos_v1.PolicyNodeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &nomos_v1.PolicyNodeList{}
	for _, item := range obj.(*nomos_v1.PolicyNodeList).Items {
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
func (c *FakePolicyNodes) Create(policyNode *nomos_v1.PolicyNode) (result *nomos_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(policynodesResource, policyNode), &nomos_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*nomos_v1.PolicyNode), err
}

// Update takes the representation of a policyNode and updates it. Returns the server's representation of the policyNode, and an error, if there is any.
func (c *FakePolicyNodes) Update(policyNode *nomos_v1.PolicyNode) (result *nomos_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(policynodesResource, policyNode), &nomos_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*nomos_v1.PolicyNode), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePolicyNodes) UpdateStatus(policyNode *nomos_v1.PolicyNode) (*nomos_v1.PolicyNode, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(policynodesResource, "status", policyNode), &nomos_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*nomos_v1.PolicyNode), err
}

// Delete takes name of the policyNode and deletes it. Returns an error if one occurs.
func (c *FakePolicyNodes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(policynodesResource, name), &nomos_v1.PolicyNode{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePolicyNodes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(policynodesResource, listOptions)

	_, err := c.Fake.Invokes(action, &nomos_v1.PolicyNodeList{})
	return err
}

// Patch applies the patch and returns the patched policyNode.
func (c *FakePolicyNodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *nomos_v1.PolicyNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(policynodesResource, name, data, subresources...), &nomos_v1.PolicyNode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*nomos_v1.PolicyNode), err
}
