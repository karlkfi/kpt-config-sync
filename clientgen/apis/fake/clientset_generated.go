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
	clientset "github.com/google/nomos/clientgen/apis"
	bespinv1 "github.com/google/nomos/clientgen/apis/typed/policyascode/v1"
	fakebespinv1 "github.com/google/nomos/clientgen/apis/typed/policyascode/v1/fake"
	nomosv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	fakenomosv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o))
	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &Clientset{fakePtr, &fakediscovery.FakeDiscovery{Fake: &fakePtr}}
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ clientset.Interface = &Clientset{}

// BespinV1 retrieves the BespinV1Client
func (c *Clientset) BespinV1() bespinv1.BespinV1Interface {
	return &fakebespinv1.FakeBespinV1{Fake: &c.Fake}
}

// Bespin retrieves the BespinV1Client
func (c *Clientset) Bespin() bespinv1.BespinV1Interface {
	return &fakebespinv1.FakeBespinV1{Fake: &c.Fake}
}

// NomosV1 retrieves the NomosV1Client
func (c *Clientset) NomosV1() nomosv1.NomosV1Interface {
	return &fakenomosv1.FakeNomosV1{Fake: &c.Fake}
}

// Nomos retrieves the NomosV1Client
func (c *Clientset) Nomos() nomosv1.NomosV1Interface {
	return &fakenomosv1.FakeNomosV1{Fake: &c.Fake}
}
