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

// PolicyNodesGetter has a method to return a PolicyNodeInterface.
// A group's client should implement this interface.
type PolicyNodesGetter interface {
	PolicyNodes() PolicyNodeInterface
}

// PolicyNodeInterface has methods to work with PolicyNode resources.
type PolicyNodeInterface interface {
	Create(*v1.PolicyNode) (*v1.PolicyNode, error)
	Update(*v1.PolicyNode) (*v1.PolicyNode, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.PolicyNode, error)
	List(opts meta_v1.ListOptions) (*v1.PolicyNodeList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PolicyNode, err error)
	PolicyNodeExpansion
}

// policyNodes implements PolicyNodeInterface
type policyNodes struct {
	client rest.Interface
}

// newPolicyNodes returns a PolicyNodes
func newPolicyNodes(c *K8usV1Client) *policyNodes {
	return &policyNodes{
		client: c.RESTClient(),
	}
}

// Get takes name of the policyNode, and returns the corresponding policyNode object, and an error if there is any.
func (c *policyNodes) Get(name string, options meta_v1.GetOptions) (result *v1.PolicyNode, err error) {
	result = &v1.PolicyNode{}
	err = c.client.Get().
		Resource("policynodes").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PolicyNodes that match those selectors.
func (c *policyNodes) List(opts meta_v1.ListOptions) (result *v1.PolicyNodeList, err error) {
	result = &v1.PolicyNodeList{}
	err = c.client.Get().
		Resource("policynodes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested policyNodes.
func (c *policyNodes) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("policynodes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a policyNode and creates it.  Returns the server's representation of the policyNode, and an error, if there is any.
func (c *policyNodes) Create(policyNode *v1.PolicyNode) (result *v1.PolicyNode, err error) {
	result = &v1.PolicyNode{}
	err = c.client.Post().
		Resource("policynodes").
		Body(policyNode).
		Do().
		Into(result)
	return
}

// Update takes the representation of a policyNode and updates it. Returns the server's representation of the policyNode, and an error, if there is any.
func (c *policyNodes) Update(policyNode *v1.PolicyNode) (result *v1.PolicyNode, err error) {
	result = &v1.PolicyNode{}
	err = c.client.Put().
		Resource("policynodes").
		Name(policyNode.Name).
		Body(policyNode).
		Do().
		Into(result)
	return
}

// Delete takes name of the policyNode and deletes it. Returns an error if one occurs.
func (c *policyNodes) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("policynodes").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *policyNodes) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("policynodes").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched policyNode.
func (c *policyNodes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.PolicyNode, err error) {
	result = &v1.PolicyNode{}
	err = c.client.Patch(pt).
		Resource("policynodes").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
