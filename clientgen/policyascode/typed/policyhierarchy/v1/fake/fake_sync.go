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
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSyncs implements SyncInterface
type FakeSyncs struct {
	Fake *FakeNomosV1
}

var syncsResource = schema.GroupVersionResource{Group: "nomos.dev", Version: "v1", Resource: "syncs"}

var syncsKind = schema.GroupVersionKind{Group: "nomos.dev", Version: "v1", Kind: "Sync"}

// Get takes name of the sync, and returns the corresponding sync object, and an error if there is any.
func (c *FakeSyncs) Get(name string, options v1.GetOptions) (result *policyhierarchy_v1.Sync, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(syncsResource, name), &policyhierarchy_v1.Sync{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.Sync), err
}

// List takes label and field selectors, and returns the list of Syncs that match those selectors.
func (c *FakeSyncs) List(opts v1.ListOptions) (result *policyhierarchy_v1.SyncList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(syncsResource, syncsKind, opts), &policyhierarchy_v1.SyncList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyhierarchy_v1.SyncList{}
	for _, item := range obj.(*policyhierarchy_v1.SyncList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested syncs.
func (c *FakeSyncs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(syncsResource, opts))
}

// Create takes the representation of a sync and creates it.  Returns the server's representation of the sync, and an error, if there is any.
func (c *FakeSyncs) Create(sync *policyhierarchy_v1.Sync) (result *policyhierarchy_v1.Sync, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(syncsResource, sync), &policyhierarchy_v1.Sync{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.Sync), err
}

// Update takes the representation of a sync and updates it. Returns the server's representation of the sync, and an error, if there is any.
func (c *FakeSyncs) Update(sync *policyhierarchy_v1.Sync) (result *policyhierarchy_v1.Sync, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(syncsResource, sync), &policyhierarchy_v1.Sync{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.Sync), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSyncs) UpdateStatus(sync *policyhierarchy_v1.Sync) (*policyhierarchy_v1.Sync, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(syncsResource, "status", sync), &policyhierarchy_v1.Sync{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.Sync), err
}

// Delete takes name of the sync and deletes it. Returns an error if one occurs.
func (c *FakeSyncs) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(syncsResource, name), &policyhierarchy_v1.Sync{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSyncs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(syncsResource, listOptions)

	_, err := c.Fake.Invokes(action, &policyhierarchy_v1.SyncList{})
	return err
}

// Patch applies the patch and returns the patched sync.
func (c *FakeSyncs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyhierarchy_v1.Sync, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(syncsResource, name, data, subresources...), &policyhierarchy_v1.Sync{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.Sync), err
}
