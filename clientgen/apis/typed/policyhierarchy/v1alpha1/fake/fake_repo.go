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

package fake

import (
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRepos implements RepoInterface
type FakeRepos struct {
	Fake *FakeNomosV1alpha1
}

var reposResource = schema.GroupVersionResource{Group: "nomos.dev", Version: "v1alpha1", Resource: "repos"}

var reposKind = schema.GroupVersionKind{Group: "nomos.dev", Version: "v1alpha1", Kind: "Repo"}

// Get takes name of the repo, and returns the corresponding repo object, and an error if there is any.
func (c *FakeRepos) Get(name string, options v1.GetOptions) (result *v1alpha1.Repo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(reposResource, name), &v1alpha1.Repo{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Repo), err
}

// List takes label and field selectors, and returns the list of Repos that match those selectors.
func (c *FakeRepos) List(opts v1.ListOptions) (result *v1alpha1.RepoList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(reposResource, reposKind, opts), &v1alpha1.RepoList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RepoList{}
	for _, item := range obj.(*v1alpha1.RepoList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested repos.
func (c *FakeRepos) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(reposResource, opts))
}

// Create takes the representation of a repo and creates it.  Returns the server's representation of the repo, and an error, if there is any.
func (c *FakeRepos) Create(repo *v1alpha1.Repo) (result *v1alpha1.Repo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(reposResource, repo), &v1alpha1.Repo{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Repo), err
}

// Update takes the representation of a repo and updates it. Returns the server's representation of the repo, and an error, if there is any.
func (c *FakeRepos) Update(repo *v1alpha1.Repo) (result *v1alpha1.Repo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(reposResource, repo), &v1alpha1.Repo{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Repo), err
}

// Delete takes name of the repo and deletes it. Returns an error if one occurs.
func (c *FakeRepos) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(reposResource, name), &v1alpha1.Repo{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRepos) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(reposResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.RepoList{})
	return err
}

// Patch applies the patch and returns the patched repo.
func (c *FakeRepos) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Repo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(reposResource, name, data, subresources...), &v1alpha1.Repo{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Repo), err
}
