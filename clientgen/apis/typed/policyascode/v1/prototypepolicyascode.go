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

package v1

import (
	scheme "github.com/google/nomos/clientgen/apis/scheme"
	v1 "github.com/google/nomos/pkg/api/policyascode/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PrototypePolicyAsCodesGetter has a method to return a PrototypePolicyAsCodeInterface.
// A group's client should implement this interface.
type PrototypePolicyAsCodesGetter interface {
	PrototypePolicyAsCodes() PrototypePolicyAsCodeInterface
}

// PrototypePolicyAsCodeInterface has methods to work with PrototypePolicyAsCode resources.
type PrototypePolicyAsCodeInterface interface {
	Create(*v1.PrototypePolicyAsCode) (*v1.PrototypePolicyAsCode, error)
	Update(*v1.PrototypePolicyAsCode) (*v1.PrototypePolicyAsCode, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.PrototypePolicyAsCode, error)
	List(opts meta_v1.ListOptions) (*v1.PrototypePolicyAsCodeList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PrototypePolicyAsCode, err error)
	PrototypePolicyAsCodeExpansion
}

// prototypePolicyAsCodes implements PrototypePolicyAsCodeInterface
type prototypePolicyAsCodes struct {
	client rest.Interface
}

// newPrototypePolicyAsCodes returns a PrototypePolicyAsCodes
func newPrototypePolicyAsCodes(c *BespinV1Client) *prototypePolicyAsCodes {
	return &prototypePolicyAsCodes{
		client: c.RESTClient(),
	}
}

// Get takes name of the prototypePolicyAsCode, and returns the corresponding prototypePolicyAsCode object, and an error if there is any.
func (c *prototypePolicyAsCodes) Get(name string, options meta_v1.GetOptions) (result *v1.PrototypePolicyAsCode, err error) {
	result = &v1.PrototypePolicyAsCode{}
	err = c.client.Get().
		Resource("prototypepolicyascodes").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PrototypePolicyAsCodes that match those selectors.
func (c *prototypePolicyAsCodes) List(opts meta_v1.ListOptions) (result *v1.PrototypePolicyAsCodeList, err error) {
	result = &v1.PrototypePolicyAsCodeList{}
	err = c.client.Get().
		Resource("prototypepolicyascodes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested prototypePolicyAsCodes.
func (c *prototypePolicyAsCodes) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("prototypepolicyascodes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a prototypePolicyAsCode and creates it.  Returns the server's representation of the prototypePolicyAsCode, and an error, if there is any.
func (c *prototypePolicyAsCodes) Create(prototypePolicyAsCode *v1.PrototypePolicyAsCode) (result *v1.PrototypePolicyAsCode, err error) {
	result = &v1.PrototypePolicyAsCode{}
	err = c.client.Post().
		Resource("prototypepolicyascodes").
		Body(prototypePolicyAsCode).
		Do().
		Into(result)
	return
}

// Update takes the representation of a prototypePolicyAsCode and updates it. Returns the server's representation of the prototypePolicyAsCode, and an error, if there is any.
func (c *prototypePolicyAsCodes) Update(prototypePolicyAsCode *v1.PrototypePolicyAsCode) (result *v1.PrototypePolicyAsCode, err error) {
	result = &v1.PrototypePolicyAsCode{}
	err = c.client.Put().
		Resource("prototypepolicyascodes").
		Name(prototypePolicyAsCode.Name).
		Body(prototypePolicyAsCode).
		Do().
		Into(result)
	return
}

// Delete takes name of the prototypePolicyAsCode and deletes it. Returns an error if one occurs.
func (c *prototypePolicyAsCodes) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("prototypepolicyascodes").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *prototypePolicyAsCodes) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("prototypepolicyascodes").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched prototypePolicyAsCode.
func (c *prototypePolicyAsCodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PrototypePolicyAsCode, err error) {
	result = &v1.PrototypePolicyAsCode{}
	err = c.client.Patch(pt).
		Resource("prototypepolicyascodes").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
