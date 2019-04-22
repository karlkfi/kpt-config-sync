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

package v1

import (
	scheme "github.com/google/nomos/clientgen/apis/scheme"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// HierarchicalQuotasGetter has a method to return a HierarchicalQuotaInterface.
// A group's client should implement this interface.
type HierarchicalQuotasGetter interface {
	HierarchicalQuotas() HierarchicalQuotaInterface
}

// HierarchicalQuotaInterface has methods to work with HierarchicalQuota resources.
type HierarchicalQuotaInterface interface {
	Create(*v1.HierarchicalQuota) (*v1.HierarchicalQuota, error)
	Update(*v1.HierarchicalQuota) (*v1.HierarchicalQuota, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.HierarchicalQuota, error)
	List(opts metav1.ListOptions) (*v1.HierarchicalQuotaList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.HierarchicalQuota, err error)
	HierarchicalQuotaExpansion
}

// hierarchicalQuotas implements HierarchicalQuotaInterface
type hierarchicalQuotas struct {
	client rest.Interface
}

// newHierarchicalQuotas returns a HierarchicalQuotas
func newHierarchicalQuotas(c *ConfigmanagementV1Client) *hierarchicalQuotas {
	return &hierarchicalQuotas{
		client: c.RESTClient(),
	}
}

// Get takes name of the hierarchicalQuota, and returns the corresponding hierarchicalQuota object, and an error if there is any.
func (c *hierarchicalQuotas) Get(name string, options metav1.GetOptions) (result *v1.HierarchicalQuota, err error) {
	result = &v1.HierarchicalQuota{}
	err = c.client.Get().
		Resource("hierarchicalquotas").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of HierarchicalQuotas that match those selectors.
func (c *hierarchicalQuotas) List(opts metav1.ListOptions) (result *v1.HierarchicalQuotaList, err error) {
	result = &v1.HierarchicalQuotaList{}
	err = c.client.Get().
		Resource("hierarchicalquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested hierarchicalQuotas.
func (c *hierarchicalQuotas) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("hierarchicalquotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a hierarchicalQuota and creates it.  Returns the server's representation of the hierarchicalQuota, and an error, if there is any.
func (c *hierarchicalQuotas) Create(hierarchicalQuota *v1.HierarchicalQuota) (result *v1.HierarchicalQuota, err error) {
	result = &v1.HierarchicalQuota{}
	err = c.client.Post().
		Resource("hierarchicalquotas").
		Body(hierarchicalQuota).
		Do().
		Into(result)
	return
}

// Update takes the representation of a hierarchicalQuota and updates it. Returns the server's representation of the hierarchicalQuota, and an error, if there is any.
func (c *hierarchicalQuotas) Update(hierarchicalQuota *v1.HierarchicalQuota) (result *v1.HierarchicalQuota, err error) {
	result = &v1.HierarchicalQuota{}
	err = c.client.Put().
		Resource("hierarchicalquotas").
		Name(hierarchicalQuota.Name).
		Body(hierarchicalQuota).
		Do().
		Into(result)
	return
}

// Delete takes name of the hierarchicalQuota and deletes it. Returns an error if one occurs.
func (c *hierarchicalQuotas) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("hierarchicalquotas").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *hierarchicalQuotas) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Resource("hierarchicalquotas").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched hierarchicalQuota.
func (c *hierarchicalQuotas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.HierarchicalQuota, err error) {
	result = &v1.HierarchicalQuota{}
	err = c.client.Patch(pt).
		Resource("hierarchicalquotas").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
