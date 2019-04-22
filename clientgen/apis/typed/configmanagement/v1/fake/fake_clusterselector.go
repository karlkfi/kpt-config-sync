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

package fake

import (
	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterSelectors implements ClusterSelectorInterface
type FakeClusterSelectors struct {
	Fake *FakeConfigmanagementV1
}

var clusterselectorsResource = schema.GroupVersionResource{Group: "configmanagement.gke.io", Version: "v1", Resource: "clusterselectors"}

var clusterselectorsKind = schema.GroupVersionKind{Group: "configmanagement.gke.io", Version: "v1", Kind: "ClusterSelector"}

// Get takes name of the clusterSelector, and returns the corresponding clusterSelector object, and an error if there is any.
func (c *FakeClusterSelectors) Get(name string, options v1.GetOptions) (result *configmanagementv1.ClusterSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusterselectorsResource, name), &configmanagementv1.ClusterSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.ClusterSelector), err
}

// List takes label and field selectors, and returns the list of ClusterSelectors that match those selectors.
func (c *FakeClusterSelectors) List(opts v1.ListOptions) (result *configmanagementv1.ClusterSelectorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusterselectorsResource, clusterselectorsKind, opts), &configmanagementv1.ClusterSelectorList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &configmanagementv1.ClusterSelectorList{ListMeta: obj.(*configmanagementv1.ClusterSelectorList).ListMeta}
	for _, item := range obj.(*configmanagementv1.ClusterSelectorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterSelectors.
func (c *FakeClusterSelectors) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusterselectorsResource, opts))
}

// Create takes the representation of a clusterSelector and creates it.  Returns the server's representation of the clusterSelector, and an error, if there is any.
func (c *FakeClusterSelectors) Create(clusterSelector *configmanagementv1.ClusterSelector) (result *configmanagementv1.ClusterSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusterselectorsResource, clusterSelector), &configmanagementv1.ClusterSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.ClusterSelector), err
}

// Update takes the representation of a clusterSelector and updates it. Returns the server's representation of the clusterSelector, and an error, if there is any.
func (c *FakeClusterSelectors) Update(clusterSelector *configmanagementv1.ClusterSelector) (result *configmanagementv1.ClusterSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusterselectorsResource, clusterSelector), &configmanagementv1.ClusterSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.ClusterSelector), err
}

// Delete takes name of the clusterSelector and deletes it. Returns an error if one occurs.
func (c *FakeClusterSelectors) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clusterselectorsResource, name), &configmanagementv1.ClusterSelector{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterSelectors) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusterselectorsResource, listOptions)

	_, err := c.Fake.Invokes(action, &configmanagementv1.ClusterSelectorList{})
	return err
}

// Patch applies the patch and returns the patched clusterSelector.
func (c *FakeClusterSelectors) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *configmanagementv1.ClusterSelector, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterselectorsResource, name, data, subresources...), &configmanagementv1.ClusterSelector{})
	if obj == nil {
		return nil, err
	}
	return obj.(*configmanagementv1.ClusterSelector), err
}
