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

package v1alpha1

import (
	scheme "github.com/google/nomos/clientgen/apis/scheme"
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NamespaceSelectorsGetter has a method to return a NamespaceSelectorInterface.
// A group's client should implement this interface.
type NamespaceSelectorsGetter interface {
	NamespaceSelectors() NamespaceSelectorInterface
}

// NamespaceSelectorInterface has methods to work with NamespaceSelector resources.
type NamespaceSelectorInterface interface {
	Create(*v1alpha1.NamespaceSelector) (*v1alpha1.NamespaceSelector, error)
	Update(*v1alpha1.NamespaceSelector) (*v1alpha1.NamespaceSelector, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.NamespaceSelector, error)
	List(opts v1.ListOptions) (*v1alpha1.NamespaceSelectorList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NamespaceSelector, err error)
	NamespaceSelectorExpansion
}

// namespaceSelectors implements NamespaceSelectorInterface
type namespaceSelectors struct {
	client rest.Interface
}

// newNamespaceSelectors returns a NamespaceSelectors
func newNamespaceSelectors(c *NomosV1alpha1Client) *namespaceSelectors {
	return &namespaceSelectors{
		client: c.RESTClient(),
	}
}

// Get takes name of the namespaceSelector, and returns the corresponding namespaceSelector object, and an error if there is any.
func (c *namespaceSelectors) Get(name string, options v1.GetOptions) (result *v1alpha1.NamespaceSelector, err error) {
	result = &v1alpha1.NamespaceSelector{}
	err = c.client.Get().
		Resource("namespaceselectors").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NamespaceSelectors that match those selectors.
func (c *namespaceSelectors) List(opts v1.ListOptions) (result *v1alpha1.NamespaceSelectorList, err error) {
	result = &v1alpha1.NamespaceSelectorList{}
	err = c.client.Get().
		Resource("namespaceselectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested namespaceSelectors.
func (c *namespaceSelectors) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("namespaceselectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a namespaceSelector and creates it.  Returns the server's representation of the namespaceSelector, and an error, if there is any.
func (c *namespaceSelectors) Create(namespaceSelector *v1alpha1.NamespaceSelector) (result *v1alpha1.NamespaceSelector, err error) {
	result = &v1alpha1.NamespaceSelector{}
	err = c.client.Post().
		Resource("namespaceselectors").
		Body(namespaceSelector).
		Do().
		Into(result)
	return
}

// Update takes the representation of a namespaceSelector and updates it. Returns the server's representation of the namespaceSelector, and an error, if there is any.
func (c *namespaceSelectors) Update(namespaceSelector *v1alpha1.NamespaceSelector) (result *v1alpha1.NamespaceSelector, err error) {
	result = &v1alpha1.NamespaceSelector{}
	err = c.client.Put().
		Resource("namespaceselectors").
		Name(namespaceSelector.Name).
		Body(namespaceSelector).
		Do().
		Into(result)
	return
}

// Delete takes name of the namespaceSelector and deletes it. Returns an error if one occurs.
func (c *namespaceSelectors) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("namespaceselectors").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *namespaceSelectors) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("namespaceselectors").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched namespaceSelector.
func (c *namespaceSelectors) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NamespaceSelector, err error) {
	result = &v1alpha1.NamespaceSelector{}
	err = c.client.Patch(pt).
		Resource("namespaceselectors").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
