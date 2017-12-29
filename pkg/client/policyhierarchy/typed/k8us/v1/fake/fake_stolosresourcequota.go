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

package fake

import (
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeStolosResourceQuotas implements StolosResourceQuotaInterface
type FakeStolosResourceQuotas struct {
	Fake *FakeK8usV1
	ns   string
}

var stolosresourcequotasResource = schema.GroupVersionResource{Group: "k8us.k8s.io", Version: "v1", Resource: "stolosresourcequotas"}

var stolosresourcequotasKind = schema.GroupVersionKind{Group: "k8us.k8s.io", Version: "v1", Kind: "StolosResourceQuota"}

// Get takes name of the stolosResourceQuota, and returns the corresponding stolosResourceQuota object, and an error if there is any.
func (c *FakeStolosResourceQuotas) Get(name string, options v1.GetOptions) (result *policyhierarchy_v1.StolosResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(stolosresourcequotasResource, c.ns, name), &policyhierarchy_v1.StolosResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosResourceQuota), err
}

// List takes label and field selectors, and returns the list of StolosResourceQuotas that match those selectors.
func (c *FakeStolosResourceQuotas) List(opts v1.ListOptions) (result *policyhierarchy_v1.StolosResourceQuotaList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(stolosresourcequotasResource, stolosresourcequotasKind, c.ns, opts), &policyhierarchy_v1.StolosResourceQuotaList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &policyhierarchy_v1.StolosResourceQuotaList{}
	for _, item := range obj.(*policyhierarchy_v1.StolosResourceQuotaList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested stolosResourceQuotas.
func (c *FakeStolosResourceQuotas) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(stolosresourcequotasResource, c.ns, opts))

}

// Create takes the representation of a stolosResourceQuota and creates it.  Returns the server's representation of the stolosResourceQuota, and an error, if there is any.
func (c *FakeStolosResourceQuotas) Create(stolosResourceQuota *policyhierarchy_v1.StolosResourceQuota) (result *policyhierarchy_v1.StolosResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(stolosresourcequotasResource, c.ns, stolosResourceQuota), &policyhierarchy_v1.StolosResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosResourceQuota), err
}

// Update takes the representation of a stolosResourceQuota and updates it. Returns the server's representation of the stolosResourceQuota, and an error, if there is any.
func (c *FakeStolosResourceQuotas) Update(stolosResourceQuota *policyhierarchy_v1.StolosResourceQuota) (result *policyhierarchy_v1.StolosResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(stolosresourcequotasResource, c.ns, stolosResourceQuota), &policyhierarchy_v1.StolosResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosResourceQuota), err
}

// Delete takes name of the stolosResourceQuota and deletes it. Returns an error if one occurs.
func (c *FakeStolosResourceQuotas) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(stolosresourcequotasResource, c.ns, name), &policyhierarchy_v1.StolosResourceQuota{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeStolosResourceQuotas) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(stolosresourcequotasResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &policyhierarchy_v1.StolosResourceQuotaList{})
	return err
}

// Patch applies the patch and returns the patched stolosResourceQuota.
func (c *FakeStolosResourceQuotas) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *policyhierarchy_v1.StolosResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(stolosresourcequotasResource, c.ns, name, data, subresources...), &policyhierarchy_v1.StolosResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*policyhierarchy_v1.StolosResourceQuota), err
}
