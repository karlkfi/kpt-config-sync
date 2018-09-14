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
	v1 "github.com/google/nomos/pkg/api/nomos/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SyncsGetter has a method to return a SyncInterface.
// A group's client should implement this interface.
type SyncsGetter interface {
	Syncs() SyncInterface
}

// SyncInterface has methods to work with Sync resources.
type SyncInterface interface {
	Create(*v1.Sync) (*v1.Sync, error)
	Update(*v1.Sync) (*v1.Sync, error)
	UpdateStatus(*v1.Sync) (*v1.Sync, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.Sync, error)
	List(opts meta_v1.ListOptions) (*v1.SyncList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Sync, err error)
	SyncExpansion
}

// syncs implements SyncInterface
type syncs struct {
	client rest.Interface
}

// newSyncs returns a Syncs
func newSyncs(c *NomosV1Client) *syncs {
	return &syncs{
		client: c.RESTClient(),
	}
}

// Get takes name of the sync, and returns the corresponding sync object, and an error if there is any.
func (c *syncs) Get(name string, options meta_v1.GetOptions) (result *v1.Sync, err error) {
	result = &v1.Sync{}
	err = c.client.Get().
		Resource("syncs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Syncs that match those selectors.
func (c *syncs) List(opts meta_v1.ListOptions) (result *v1.SyncList, err error) {
	result = &v1.SyncList{}
	err = c.client.Get().
		Resource("syncs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested syncs.
func (c *syncs) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("syncs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a sync and creates it.  Returns the server's representation of the sync, and an error, if there is any.
func (c *syncs) Create(sync *v1.Sync) (result *v1.Sync, err error) {
	result = &v1.Sync{}
	err = c.client.Post().
		Resource("syncs").
		Body(sync).
		Do().
		Into(result)
	return
}

// Update takes the representation of a sync and updates it. Returns the server's representation of the sync, and an error, if there is any.
func (c *syncs) Update(sync *v1.Sync) (result *v1.Sync, err error) {
	result = &v1.Sync{}
	err = c.client.Put().
		Resource("syncs").
		Name(sync.Name).
		Body(sync).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *syncs) UpdateStatus(sync *v1.Sync) (result *v1.Sync, err error) {
	result = &v1.Sync{}
	err = c.client.Put().
		Resource("syncs").
		Name(sync.Name).
		SubResource("status").
		Body(sync).
		Do().
		Into(result)
	return
}

// Delete takes name of the sync and deletes it. Returns an error if one occurs.
func (c *syncs) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("syncs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *syncs) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("syncs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched sync.
func (c *syncs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Sync, err error) {
	result = &v1.Sync{}
	err = c.client.Patch(pt).
		Resource("syncs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
