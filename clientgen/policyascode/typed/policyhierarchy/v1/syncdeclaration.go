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
	scheme "github.com/google/nomos/clientgen/policyascode/scheme"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SyncDeclarationsGetter has a method to return a SyncDeclarationInterface.
// A group's client should implement this interface.
type SyncDeclarationsGetter interface {
	SyncDeclarations() SyncDeclarationInterface
}

// SyncDeclarationInterface has methods to work with SyncDeclaration resources.
type SyncDeclarationInterface interface {
	Create(*v1.SyncDeclaration) (*v1.SyncDeclaration, error)
	Update(*v1.SyncDeclaration) (*v1.SyncDeclaration, error)
	UpdateStatus(*v1.SyncDeclaration) (*v1.SyncDeclaration, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.SyncDeclaration, error)
	List(opts meta_v1.ListOptions) (*v1.SyncDeclarationList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SyncDeclaration, err error)
	SyncDeclarationExpansion
}

// syncDeclarations implements SyncDeclarationInterface
type syncDeclarations struct {
	client rest.Interface
}

// newSyncDeclarations returns a SyncDeclarations
func newSyncDeclarations(c *NomosV1Client) *syncDeclarations {
	return &syncDeclarations{
		client: c.RESTClient(),
	}
}

// Get takes name of the syncDeclaration, and returns the corresponding syncDeclaration object, and an error if there is any.
func (c *syncDeclarations) Get(name string, options meta_v1.GetOptions) (result *v1.SyncDeclaration, err error) {
	result = &v1.SyncDeclaration{}
	err = c.client.Get().
		Resource("syncdeclarations").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SyncDeclarations that match those selectors.
func (c *syncDeclarations) List(opts meta_v1.ListOptions) (result *v1.SyncDeclarationList, err error) {
	result = &v1.SyncDeclarationList{}
	err = c.client.Get().
		Resource("syncdeclarations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested syncDeclarations.
func (c *syncDeclarations) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("syncdeclarations").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a syncDeclaration and creates it.  Returns the server's representation of the syncDeclaration, and an error, if there is any.
func (c *syncDeclarations) Create(syncDeclaration *v1.SyncDeclaration) (result *v1.SyncDeclaration, err error) {
	result = &v1.SyncDeclaration{}
	err = c.client.Post().
		Resource("syncdeclarations").
		Body(syncDeclaration).
		Do().
		Into(result)
	return
}

// Update takes the representation of a syncDeclaration and updates it. Returns the server's representation of the syncDeclaration, and an error, if there is any.
func (c *syncDeclarations) Update(syncDeclaration *v1.SyncDeclaration) (result *v1.SyncDeclaration, err error) {
	result = &v1.SyncDeclaration{}
	err = c.client.Put().
		Resource("syncdeclarations").
		Name(syncDeclaration.Name).
		Body(syncDeclaration).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *syncDeclarations) UpdateStatus(syncDeclaration *v1.SyncDeclaration) (result *v1.SyncDeclaration, err error) {
	result = &v1.SyncDeclaration{}
	err = c.client.Put().
		Resource("syncdeclarations").
		Name(syncDeclaration.Name).
		SubResource("status").
		Body(syncDeclaration).
		Do().
		Into(result)
	return
}

// Delete takes name of the syncDeclaration and deletes it. Returns an error if one occurs.
func (c *syncDeclarations) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("syncdeclarations").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *syncDeclarations) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("syncdeclarations").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched syncDeclaration.
func (c *syncDeclarations) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.SyncDeclaration, err error) {
	result = &v1.SyncDeclaration{}
	err = c.client.Patch(pt).
		Resource("syncdeclarations").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
