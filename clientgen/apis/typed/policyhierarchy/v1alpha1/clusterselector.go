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

// ClusterSelectorsGetter has a method to return a ClusterSelectorInterface.
// A group's client should implement this interface.
type ClusterSelectorsGetter interface {
	ClusterSelectors() ClusterSelectorInterface
}

// ClusterSelectorInterface has methods to work with ClusterSelector resources.
type ClusterSelectorInterface interface {
	Create(*v1alpha1.ClusterSelector) (*v1alpha1.ClusterSelector, error)
	Update(*v1alpha1.ClusterSelector) (*v1alpha1.ClusterSelector, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ClusterSelector, error)
	List(opts v1.ListOptions) (*v1alpha1.ClusterSelectorList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterSelector, err error)
	ClusterSelectorExpansion
}

// clusterSelectors implements ClusterSelectorInterface
type clusterSelectors struct {
	client rest.Interface
}

// newClusterSelectors returns a ClusterSelectors
func newClusterSelectors(c *NomosV1alpha1Client) *clusterSelectors {
	return &clusterSelectors{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterSelector, and returns the corresponding clusterSelector object, and an error if there is any.
func (c *clusterSelectors) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterSelector, err error) {
	result = &v1alpha1.ClusterSelector{}
	err = c.client.Get().
		Resource("clusterselectors").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterSelectors that match those selectors.
func (c *clusterSelectors) List(opts v1.ListOptions) (result *v1alpha1.ClusterSelectorList, err error) {
	result = &v1alpha1.ClusterSelectorList{}
	err = c.client.Get().
		Resource("clusterselectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterSelectors.
func (c *clusterSelectors) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("clusterselectors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a clusterSelector and creates it.  Returns the server's representation of the clusterSelector, and an error, if there is any.
func (c *clusterSelectors) Create(clusterSelector *v1alpha1.ClusterSelector) (result *v1alpha1.ClusterSelector, err error) {
	result = &v1alpha1.ClusterSelector{}
	err = c.client.Post().
		Resource("clusterselectors").
		Body(clusterSelector).
		Do().
		Into(result)
	return
}

// Update takes the representation of a clusterSelector and updates it. Returns the server's representation of the clusterSelector, and an error, if there is any.
func (c *clusterSelectors) Update(clusterSelector *v1alpha1.ClusterSelector) (result *v1alpha1.ClusterSelector, err error) {
	result = &v1alpha1.ClusterSelector{}
	err = c.client.Put().
		Resource("clusterselectors").
		Name(clusterSelector.Name).
		Body(clusterSelector).
		Do().
		Into(result)
	return
}

// Delete takes name of the clusterSelector and deletes it. Returns an error if one occurs.
func (c *clusterSelectors) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterselectors").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterSelectors) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("clusterselectors").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched clusterSelector.
func (c *clusterSelectors) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterSelector, err error) {
	result = &v1alpha1.ClusterSelector{}
	err = c.client.Patch(pt).
		Resource("clusterselectors").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
