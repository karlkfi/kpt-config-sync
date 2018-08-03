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
	policyascode_v1 "github.com/google/nomos/pkg/api/policyascode/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePrototypePolicyAsCodes implements PrototypePolicyAsCodeInterface
type FakePrototypePolicyAsCodes struct {
	Fake *FakeBespinV1
}

var prototypepolicyascodesResource = schema.GroupVersionResource{Group: "bespin.dev", Version: "v1", Resource: "prototypepolicyascodes"}

var prototypepolicyascodesKind = schema.GroupVersionKind{Group: "bespin.dev", Version: "v1", Kind: "PrototypePolicyAsCode"}

// Get takes name of the prototypePolicyAsCode, and returns the corresponding prototypePolicyAsCode object, and an error if there is any.
func (c *FakePrototypePolicyAsCodes) Get(name string, options v1.GetOptions) (result *policyascode_v1.PrototypePolicyAsCode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(prototypepolicyascodesResource, name), &policyascode_v1.PrototypePolicyAsCode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.PrototypePolicyAsCode), err
}

// List takes label and field selectors, and returns the list of PrototypePolicyAsCodes that match those selectors.
func (c *FakePrototypePolicyAsCodes) List(opts v1.ListOptions) (result *policyascode_v1.PrototypePolicyAsCodeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(prototypepolicyascodesResource, prototypepolicyascodesKind, opts), &policyascode_v1.PrototypePolicyAsCodeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyascode_v1.PrototypePolicyAsCodeList{}
	for _, item := range obj.(*policyascode_v1.PrototypePolicyAsCodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested prototypePolicyAsCodes.
func (c *FakePrototypePolicyAsCodes) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(prototypepolicyascodesResource, opts))
}

// Create takes the representation of a prototypePolicyAsCode and creates it.  Returns the server's representation of the prototypePolicyAsCode, and an error, if there is any.
func (c *FakePrototypePolicyAsCodes) Create(prototypePolicyAsCode *policyascode_v1.PrototypePolicyAsCode) (result *policyascode_v1.PrototypePolicyAsCode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(prototypepolicyascodesResource, prototypePolicyAsCode), &policyascode_v1.PrototypePolicyAsCode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.PrototypePolicyAsCode), err
}

// Update takes the representation of a prototypePolicyAsCode and updates it. Returns the server's representation of the prototypePolicyAsCode, and an error, if there is any.
func (c *FakePrototypePolicyAsCodes) Update(prototypePolicyAsCode *policyascode_v1.PrototypePolicyAsCode) (result *policyascode_v1.PrototypePolicyAsCode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(prototypepolicyascodesResource, prototypePolicyAsCode), &policyascode_v1.PrototypePolicyAsCode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.PrototypePolicyAsCode), err
}

// Delete takes name of the prototypePolicyAsCode and deletes it. Returns an error if one occurs.
func (c *FakePrototypePolicyAsCodes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(prototypepolicyascodesResource, name), &policyascode_v1.PrototypePolicyAsCode{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePrototypePolicyAsCodes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(prototypepolicyascodesResource, listOptions)

	_, err := c.Fake.Invokes(action, &policyascode_v1.PrototypePolicyAsCodeList{})
	return err
}

// Patch applies the patch and returns the patched prototypePolicyAsCode.
func (c *FakePrototypePolicyAsCodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyascode_v1.PrototypePolicyAsCode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(prototypepolicyascodesResource, name, data, subresources...), &policyascode_v1.PrototypePolicyAsCode{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyascode_v1.PrototypePolicyAsCode), err
}
