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

// FakeSyncDeclarations implements SyncDeclarationInterface
type FakeSyncDeclarations struct {
	Fake *FakeNomosV1
}

var syncdeclarationsResource = schema.GroupVersionResource{Group: "nomos.dev", Version: "v1", Resource: "syncdeclarations"}

var syncdeclarationsKind = schema.GroupVersionKind{Group: "nomos.dev", Version: "v1", Kind: "SyncDeclaration"}

// Get takes name of the syncDeclaration, and returns the corresponding syncDeclaration object, and an error if there is any.
func (c *FakeSyncDeclarations) Get(name string, options v1.GetOptions) (result *policyhierarchy_v1.SyncDeclaration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(syncdeclarationsResource, name), &policyhierarchy_v1.SyncDeclaration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.SyncDeclaration), err
}

// List takes label and field selectors, and returns the list of SyncDeclarations that match those selectors.
func (c *FakeSyncDeclarations) List(opts v1.ListOptions) (result *policyhierarchy_v1.SyncDeclarationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(syncdeclarationsResource, syncdeclarationsKind, opts), &policyhierarchy_v1.SyncDeclarationList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyhierarchy_v1.SyncDeclarationList{}
	for _, item := range obj.(*policyhierarchy_v1.SyncDeclarationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested syncDeclarations.
func (c *FakeSyncDeclarations) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(syncdeclarationsResource, opts))
}

// Create takes the representation of a syncDeclaration and creates it.  Returns the server's representation of the syncDeclaration, and an error, if there is any.
func (c *FakeSyncDeclarations) Create(syncDeclaration *policyhierarchy_v1.SyncDeclaration) (result *policyhierarchy_v1.SyncDeclaration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(syncdeclarationsResource, syncDeclaration), &policyhierarchy_v1.SyncDeclaration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.SyncDeclaration), err
}

// Update takes the representation of a syncDeclaration and updates it. Returns the server's representation of the syncDeclaration, and an error, if there is any.
func (c *FakeSyncDeclarations) Update(syncDeclaration *policyhierarchy_v1.SyncDeclaration) (result *policyhierarchy_v1.SyncDeclaration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(syncdeclarationsResource, syncDeclaration), &policyhierarchy_v1.SyncDeclaration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.SyncDeclaration), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSyncDeclarations) UpdateStatus(syncDeclaration *policyhierarchy_v1.SyncDeclaration) (*policyhierarchy_v1.SyncDeclaration, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(syncdeclarationsResource, "status", syncDeclaration), &policyhierarchy_v1.SyncDeclaration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.SyncDeclaration), err
}

// Delete takes name of the syncDeclaration and deletes it. Returns an error if one occurs.
func (c *FakeSyncDeclarations) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(syncdeclarationsResource, name), &policyhierarchy_v1.SyncDeclaration{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSyncDeclarations) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(syncdeclarationsResource, listOptions)

	_, err := c.Fake.Invokes(action, &policyhierarchy_v1.SyncDeclarationList{})
	return err
}

// Patch applies the patch and returns the patched syncDeclaration.
func (c *FakeSyncDeclarations) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyhierarchy_v1.SyncDeclaration, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(syncdeclarationsResource, name, data, subresources...), &policyhierarchy_v1.SyncDeclaration{})
	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.SyncDeclaration), err
}
