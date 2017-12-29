/*
Copyright 2017 The Stolos Authors.

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

// StolosResourceQuotasGetter has a method to return a StolosResourceQuotaInterface.
// A group's client should implement this interface.
type StolosResourceQuotasGetter interface {
	StolosResourceQuotas(namespace string) StolosResourceQuotaInterface
}

// StolosResourceQuotaInterface has methods to work with StolosResourceQuota resources.
type StolosResourceQuotaInterface interface {
	Create(*v1.StolosResourceQuota) (*v1.StolosResourceQuota, error)
	Update(*v1.StolosResourceQuota) (*v1.StolosResourceQuota, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.StolosResourceQuota, error)
	List(opts meta_v1.ListOptions) (*v1.StolosResourceQuotaList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.StolosResourceQuota, err error)
	StolosResourceQuotaExpansion
}

// stolosResourceQuotas implements StolosResourceQuotaInterface
type stolosResourceQuotas struct {
	client rest.Interface
	ns     string
}

// newStolosResourceQuotas returns a StolosResourceQuotas
func newStolosResourceQuotas(c *K8usV1Client, namespace string) *stolosResourceQuotas {
	return &stolosResourceQuotas{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the stolosResourceQuota, and returns the corresponding stolosResourceQuota object, and an error if there is any.
func (c *stolosResourceQuotas) Get(name string, options meta_v1.GetOptions) (result *v1.StolosResourceQuota, err error) {
	result = &v1.StolosResourceQuota{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of StolosResourceQuotas that match those selectors.
func (c *stolosResourceQuotas) List(opts meta_v1.ListOptions) (result *v1.StolosResourceQuotaList, err error) {
	result = &v1.StolosResourceQuotaList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested stolosResourceQuotas.
func (c *stolosResourceQuotas) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a stolosResourceQuota and creates it.  Returns the server's representation of the stolosResourceQuota, and an error, if there is any.
func (c *stolosResourceQuotas) Create(stolosResourceQuota *v1.StolosResourceQuota) (result *v1.StolosResourceQuota, err error) {
	result = &v1.StolosResourceQuota{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		Body(stolosResourceQuota).
		Do().
		Into(result)
	return
}

// Update takes the representation of a stolosResourceQuota and updates it. Returns the server's representation of the stolosResourceQuota, and an error, if there is any.
func (c *stolosResourceQuotas) Update(stolosResourceQuota *v1.StolosResourceQuota) (result *v1.StolosResourceQuota, err error) {
	result = &v1.StolosResourceQuota{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		Name(stolosResourceQuota.Name).
		Body(stolosResourceQuota).
		Do().
		Into(result)
	return
}

// Delete takes name of the stolosResourceQuota and deletes it. Returns an error if one occurs.
func (c *stolosResourceQuotas) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *stolosResourceQuotas) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched stolosResourceQuota.
func (c *stolosResourceQuotas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.StolosResourceQuota, err error) {
	result = &v1.StolosResourceQuota{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("stolosresourcequotas").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
