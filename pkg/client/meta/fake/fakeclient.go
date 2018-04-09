/*
Copyright 2017 The Nomos Authors.

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

// Package fake implements a fake meta.Client
package fake

import (
	"github.com/google/nomos/clientgen/policyhierarchy"
	fakepolicyhierarchy "github.com/google/nomos/clientgen/policyhierarchy/fake"
	"github.com/google/nomos/pkg/client/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
)

// Client implements meta.Interface with fake clientsets.
type Client struct {
	KubernetesClientset      *fakekubernetes.Clientset
	PolicyhierarchyClientset *fakepolicyhierarchy.Clientset
}

var _ meta.Interface = &Client{}

// NewClient creates a FakeClient with default simple clientsets and empty
// storage.
func NewClient() *Client {
	empty := []runtime.Object{}
	return NewClientWithStorage(empty, empty)
}

// NewClientWithStorage creates a fake meta-client and injects objects from
// kubernetesStorage as kubernetes objects, and policyHierarchyStorage as
// objects from policy hierarchy.
//
// Note, with some additional registration it may be possible to unify the two
// parameters into just one, but it's probably less hassle for the tests to
// punt on that and simply keep the two fake stores separate.
func NewClientWithStorage(kubernetesStorage, policyHierarchyStorage []runtime.Object) *Client {
	return &Client{
		KubernetesClientset:      fakekubernetes.NewSimpleClientset(kubernetesStorage...),
		PolicyhierarchyClientset: fakepolicyhierarchy.NewSimpleClientset(policyHierarchyStorage...),
	}
}

// Kubernetes implements meta.Interface
func (c *Client) Kubernetes() kubernetes.Interface {
	return c.KubernetesClientset
}

// PolicyHierarchy implements meta.Interface
func (c *Client) PolicyHierarchy() policyhierarchy.Interface {
	return c.PolicyhierarchyClientset
}
