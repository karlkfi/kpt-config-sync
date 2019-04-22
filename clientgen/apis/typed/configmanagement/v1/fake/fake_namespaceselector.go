/*
Copyright 2019 The CSP Config Management Authors.

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
	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeNamespaceSelectors implements NamespaceSelectorInterface
type FakeNamespaceSelectors struct {
	Fake *FakeConfigmanagementV1
}

var namespaceselectorsResource = schema.GroupVersionResource{Group: "configmanagement.gke.io", Version: "v1", Resource: "namespaceselectors"}

var namespaceselectorsKind = schema.GroupVersionKind{Group: "configmanagement.gke.io", Version: "v1", Kind: "NamespaceSelector"}

// Get takes name of the namespaceSelector, and returns the corresponding namespaceSelector object, and an error if there is any.
func (c *FakeNamespaceSelectors) Get(name string, options v1.GetOptions) (result *configmanagementv1.NamespaceSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(namespaceselectorsResource, name), &configmanagementv1.NamespaceSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.NamespaceSelector), err
}

// List takes label and field selectors, and returns the list of NamespaceSelectors that match those selectors.
func (c *FakeNamespaceSelectors) List(opts v1.ListOptions) (result *configmanagementv1.NamespaceSelectorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(namespaceselectorsResource, namespaceselectorsKind, opts), &configmanagementv1.NamespaceSelectorList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &configmanagementv1.NamespaceSelectorList{ListMeta: obj.(*configmanagementv1.NamespaceSelectorList).ListMeta}
	for _, item := range obj.(*configmanagementv1.NamespaceSelectorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested namespaceSelectors.
func (c *FakeNamespaceSelectors) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(namespaceselectorsResource, opts))
}

// Create takes the representation of a namespaceSelector and creates it.  Returns the server's representation of the namespaceSelector, and an error, if there is any.
func (c *FakeNamespaceSelectors) Create(namespaceSelector *configmanagementv1.NamespaceSelector) (result *configmanagementv1.NamespaceSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(namespaceselectorsResource, namespaceSelector), &configmanagementv1.NamespaceSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.NamespaceSelector), err
}

// Update takes the representation of a namespaceSelector and updates it. Returns the server's representation of the namespaceSelector, and an error, if there is any.
func (c *FakeNamespaceSelectors) Update(namespaceSelector *configmanagementv1.NamespaceSelector) (result *configmanagementv1.NamespaceSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(namespaceselectorsResource, namespaceSelector), &configmanagementv1.NamespaceSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.NamespaceSelector), err
}

// Delete takes name of the namespaceSelector and deletes it. Returns an error if one occurs.
func (c *FakeNamespaceSelectors) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(namespaceselectorsResource, name), &configmanagementv1.NamespaceSelector{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNamespaceSelectors) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(namespaceselectorsResource, listOptions)

	_, err := c.Fake.Invokes(action, &configmanagementv1.NamespaceSelectorList{})
	return err
}

// Patch applies the patch and returns the patched namespaceSelector.
func (c *FakeNamespaceSelectors) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *configmanagementv1.NamespaceSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(namespaceselectorsResource, name, data, subresources...), &configmanagementv1.NamespaceSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.NamespaceSelector), err
}
