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
	"reflect"
	"time"

	"github.com/google/nomos/clientgen/apis"
	fakepolicyhierarchy "github.com/google/nomos/clientgen/apis/fake"
	phinformers "github.com/google/nomos/clientgen/informer"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/nomos/v1"
	"github.com/google/nomos/pkg/client/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
)

// Client implements meta.Interface with fake clientsets.
type Client struct {
	KubernetesClientset      *fakekubernetes.Clientset
	PolicyhierarchyClientset *fakepolicyhierarchy.Clientset

	PolicyHierarchyInformers phinformers.SharedInformerFactory
	KubernetesInformers      informers.SharedInformerFactory
	ResyncPeriod             time.Duration
}

var _ meta.Interface = &Client{}

// NewClient creates a FakeClient with default simple clientsets and empty
// storage.
func NewClient() *Client {
	return NewClientWithStorage([]runtime.Object{})
}

// NewClientWithStorage creates a fake meta-client and injects objects from
// kubernetesStorage as kubernetes objects, and policyHierarchyStorage as
// objects from policy hierarchy.
func NewClientWithStorage(storage []runtime.Object) *Client {
	scheme := runtime.NewScheme()
	if err := policyhierarchy_v1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	phTypes := map[reflect.Type]bool{}
	for gvk, t := range scheme.AllKnownTypes() {
		if gvk.Group != policyhierarchy_v1.SchemeGroupVersion.Group {
			continue
		}
		phTypes[t] = true
	}

	var kubernetesStorage, policyHierarchyStorage []runtime.Object
	for _, obj := range storage {
		if phTypes[reflect.TypeOf(obj).Elem()] {
			policyHierarchyStorage = append(policyHierarchyStorage, obj)
		} else {
			kubernetesStorage = append(kubernetesStorage, obj)
		}
	}

	kubernetesClientset := fakekubernetes.NewSimpleClientset(kubernetesStorage...)
	policyhierarchyClientset := fakepolicyhierarchy.NewSimpleClientset(policyHierarchyStorage...)
	return &Client{
		KubernetesClientset:      kubernetesClientset,
		PolicyhierarchyClientset: policyhierarchyClientset,
		KubernetesInformers:      informers.NewSharedInformerFactory(kubernetesClientset, time.Second*2),
		PolicyHierarchyInformers: phinformers.NewSharedInformerFactory(policyhierarchyClientset, time.Second*2),
	}
}

// Kubernetes implements meta.Interface
func (c *Client) Kubernetes() kubernetes.Interface {
	return c.KubernetesClientset
}

// PolicyHierarchy implements meta.Interface
func (c *Client) PolicyHierarchy() apis.Interface {
	return c.PolicyhierarchyClientset
}
