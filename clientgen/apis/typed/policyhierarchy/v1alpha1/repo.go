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

// ReposGetter has a method to return a RepoInterface.
// A group's client should implement this interface.
type ReposGetter interface {
	Repos() RepoInterface
}

// RepoInterface has methods to work with Repo resources.
type RepoInterface interface {
	Create(*v1alpha1.Repo) (*v1alpha1.Repo, error)
	Update(*v1alpha1.Repo) (*v1alpha1.Repo, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Repo, error)
	List(opts v1.ListOptions) (*v1alpha1.RepoList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Repo, err error)
	RepoExpansion
}

// repos implements RepoInterface
type repos struct {
	client rest.Interface
}

// newRepos returns a Repos
func newRepos(c *NomosV1alpha1Client) *repos {
	return &repos{
		client: c.RESTClient(),
	}
}

// Get takes name of the repo, and returns the corresponding repo object, and an error if there is any.
func (c *repos) Get(name string, options v1.GetOptions) (result *v1alpha1.Repo, err error) {
	result = &v1alpha1.Repo{}
	err = c.client.Get().
		Resource("repos").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Repos that match those selectors.
func (c *repos) List(opts v1.ListOptions) (result *v1alpha1.RepoList, err error) {
	result = &v1alpha1.RepoList{}
	err = c.client.Get().
		Resource("repos").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested repos.
func (c *repos) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("repos").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a repo and creates it.  Returns the server's representation of the repo, and an error, if there is any.
func (c *repos) Create(repo *v1alpha1.Repo) (result *v1alpha1.Repo, err error) {
	result = &v1alpha1.Repo{}
	err = c.client.Post().
		Resource("repos").
		Body(repo).
		Do().
		Into(result)
	return
}

// Update takes the representation of a repo and updates it. Returns the server's representation of the repo, and an error, if there is any.
func (c *repos) Update(repo *v1alpha1.Repo) (result *v1alpha1.Repo, err error) {
	result = &v1alpha1.Repo{}
	err = c.client.Put().
		Resource("repos").
		Name(repo.Name).
		Body(repo).
		Do().
		Into(result)
	return
}

// Delete takes name of the repo and deletes it. Returns an error if one occurs.
func (c *repos) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("repos").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *repos) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("repos").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched repo.
func (c *repos) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Repo, err error) {
	result = &v1alpha1.Repo{}
	err = c.client.Patch(pt).
		Resource("repos").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
