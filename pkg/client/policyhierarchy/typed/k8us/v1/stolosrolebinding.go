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

package v1

import (
	v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	scheme "github.com/google/stolos/pkg/client/policyhierarchy/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// StolosRoleBindingsGetter has a method to return a StolosRoleBindingInterface.
// A group's client should implement this interface.
type StolosRoleBindingsGetter interface {
	StolosRoleBindings(namespace string) StolosRoleBindingInterface
}

// StolosRoleBindingInterface has methods to work with StolosRoleBinding resources.
type StolosRoleBindingInterface interface {
	Create(*v1.StolosRoleBinding) (*v1.StolosRoleBinding, error)
	Update(*v1.StolosRoleBinding) (*v1.StolosRoleBinding, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.StolosRoleBinding, error)
	List(opts meta_v1.ListOptions) (*v1.StolosRoleBindingList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.StolosRoleBinding, err error)
	StolosRoleBindingExpansion
}

// stolosRoleBindings implements StolosRoleBindingInterface
type stolosRoleBindings struct {
	client rest.Interface
	ns     string
}

// newStolosRoleBindings returns a StolosRoleBindings
func newStolosRoleBindings(c *K8usV1Client, namespace string) *stolosRoleBindings {
	return &stolosRoleBindings{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the stolosRoleBinding, and returns the corresponding stolosRoleBinding object, and an error if there is any.
func (c *stolosRoleBindings) Get(name string, options meta_v1.GetOptions) (result *v1.StolosRoleBinding, err error) {
	result = &v1.StolosRoleBinding{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of StolosRoleBindings that match those selectors.
func (c *stolosRoleBindings) List(opts meta_v1.ListOptions) (result *v1.StolosRoleBindingList, err error) {
	result = &v1.StolosRoleBindingList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested stolosRoleBindings.
func (c *stolosRoleBindings) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a stolosRoleBinding and creates it.  Returns the server's representation of the stolosRoleBinding, and an error, if there is any.
func (c *stolosRoleBindings) Create(stolosRoleBinding *v1.StolosRoleBinding) (result *v1.StolosRoleBinding, err error) {
	result = &v1.StolosRoleBinding{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		Body(stolosRoleBinding).
		Do().
		Into(result)
	return
}

// Update takes the representation of a stolosRoleBinding and updates it. Returns the server's representation of the stolosRoleBinding, and an error, if there is any.
func (c *stolosRoleBindings) Update(stolosRoleBinding *v1.StolosRoleBinding) (result *v1.StolosRoleBinding, err error) {
	result = &v1.StolosRoleBinding{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		Name(stolosRoleBinding.Name).
		Body(stolosRoleBinding).
		Do().
		Into(result)
	return
}

// Delete takes name of the stolosRoleBinding and deletes it. Returns an error if one occurs.
func (c *stolosRoleBindings) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *stolosRoleBindings) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stolosrolebindings").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched stolosRoleBinding.
func (c *stolosRoleBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.StolosRoleBinding, err error) {
	result = &v1.StolosRoleBinding{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("stolosrolebindings").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
