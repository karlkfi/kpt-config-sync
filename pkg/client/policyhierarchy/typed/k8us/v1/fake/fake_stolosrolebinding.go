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

// FakeStolosRoleBindings implements StolosRoleBindingInterface
type FakeStolosRoleBindings struct {
	Fake *FakeK8usV1
	ns   string
}

var stolosrolebindingsResource = schema.GroupVersionResource{Group: "k8us.k8s.io", Version: "v1", Resource: "stolosrolebindings"}

var stolosrolebindingsKind = schema.GroupVersionKind{Group: "k8us.k8s.io", Version: "v1", Kind: "StolosRoleBinding"}

// Get takes name of the stolosRoleBinding, and returns the corresponding stolosRoleBinding object, and an error if there is any.
func (c *FakeStolosRoleBindings) Get(name string, options v1.GetOptions) (result *policyhierarchy_v1.StolosRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(stolosrolebindingsResource, c.ns, name), &policyhierarchy_v1.StolosRoleBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosRoleBinding), err
}

// List takes label and field selectors, and returns the list of StolosRoleBindings that match those selectors.
func (c *FakeStolosRoleBindings) List(opts v1.ListOptions) (result *policyhierarchy_v1.StolosRoleBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(stolosrolebindingsResource, stolosrolebindingsKind, c.ns, opts), &policyhierarchy_v1.StolosRoleBindingList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyhierarchy_v1.StolosRoleBindingList{}
	for _, item := range obj.(*policyhierarchy_v1.StolosRoleBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested stolosRoleBindings.
func (c *FakeStolosRoleBindings) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(stolosrolebindingsResource, c.ns, opts))

}

// Create takes the representation of a stolosRoleBinding and creates it.  Returns the server's representation of the stolosRoleBinding, and an error, if there is any.
func (c *FakeStolosRoleBindings) Create(stolosRoleBinding *policyhierarchy_v1.StolosRoleBinding) (result *policyhierarchy_v1.StolosRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(stolosrolebindingsResource, c.ns, stolosRoleBinding), &policyhierarchy_v1.StolosRoleBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosRoleBinding), err
}

// Update takes the representation of a stolosRoleBinding and updates it. Returns the server's representation of the stolosRoleBinding, and an error, if there is any.
func (c *FakeStolosRoleBindings) Update(stolosRoleBinding *policyhierarchy_v1.StolosRoleBinding) (result *policyhierarchy_v1.StolosRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(stolosrolebindingsResource, c.ns, stolosRoleBinding), &policyhierarchy_v1.StolosRoleBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosRoleBinding), err
}

// Delete takes name of the stolosRoleBinding and deletes it. Returns an error if one occurs.
func (c *FakeStolosRoleBindings) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(stolosrolebindingsResource, c.ns, name), &policyhierarchy_v1.StolosRoleBinding{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeStolosRoleBindings) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(stolosrolebindingsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &policyhierarchy_v1.StolosRoleBindingList{})
	return err
}

// Patch applies the patch and returns the patched stolosRoleBinding.
func (c *FakeStolosRoleBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyhierarchy_v1.StolosRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(stolosrolebindingsResource, c.ns, name, data, subresources...), &policyhierarchy_v1.StolosRoleBinding{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosRoleBinding), err
}
