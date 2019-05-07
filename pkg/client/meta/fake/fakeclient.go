/*
Copyright 2017 The CSP Config Management Authors.

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
	fakeconfigmanagement "github.com/google/nomos/clientgen/apis/fake"
	cminformers "github.com/google/nomos/clientgen/informer"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/meta"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	fakeapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client implements meta.Interface with fake clientsets.
type Client struct {
	KubernetesClientset       *fakekubernetes.Clientset
	ConfigManagementClientset *fakeconfigmanagement.Clientset
	APIExtensionsClientset    *fakeapiextensions.Clientset
	RuntimeClient             client.Client

	ConfigManagementInformers cminformers.SharedInformerFactory
	KubernetesInformers       informers.SharedInformerFactory
	ResyncPeriod              time.Duration
}

var _ meta.Interface = &Client{}

// NewClient creates a FakeClient with default simple clientsets and empty
// storage.
func NewClient(runtimeClient client.Client) *Client {
	return NewClientWithStorage([]runtime.Object{}, runtimeClient)
}

// NewClientWithStorage creates a fake meta-client and injects objects from
// kubernetesStorage as kubernetes objects, and configManagementStorage as
// objects from config hierarchy.
func NewClientWithStorage(storage []runtime.Object, runtimeClient client.Client) *Client {
	scheme := runtime.NewScheme()
	if err := v1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	cmTypes := map[reflect.Type]bool{}
	for gvk, t := range scheme.AllKnownTypes() {
		if gvk.Group != v1.SchemeGroupVersion.Group {
			continue
		}
		cmTypes[t] = true
	}

	var kubernetesStorage, configManagementStorage []runtime.Object
	for _, obj := range storage {
		if cmTypes[reflect.TypeOf(obj).Elem()] {
			configManagementStorage = append(configManagementStorage, obj)
		} else {
			kubernetesStorage = append(kubernetesStorage, obj)
		}
	}

	kubernetesClientset := fakekubernetes.NewSimpleClientset(kubernetesStorage...)
	configmanagementClientset := fakeconfigmanagement.NewSimpleClientset(configManagementStorage...)
	apiExtensionsClientset := fakeapiextensions.NewSimpleClientset()
	return &Client{
		KubernetesClientset:       kubernetesClientset,
		ConfigManagementClientset: configmanagementClientset,
		APIExtensionsClientset:    apiExtensionsClientset,
		RuntimeClient:             runtimeClient,
		KubernetesInformers:       informers.NewSharedInformerFactory(kubernetesClientset, time.Second*2),
		ConfigManagementInformers: cminformers.NewSharedInformerFactory(configmanagementClientset, time.Second*2),
	}
}

// Kubernetes implements meta.Interface
func (c *Client) Kubernetes() kubernetes.Interface {
	return c.KubernetesClientset
}

// ConfigManagement implements meta.Interface
func (c *Client) ConfigManagement() apis.Interface {
	return c.ConfigManagementClientset
}

// APIExtensions implements meta.Interface
func (c *Client) APIExtensions() apiextensions.Interface {
	return c.APIExtensionsClientset
}

// Runtime returns the kubernetes runtime client for CRUD operations.
func (c *Client) Runtime() client.Client {
	return c.RuntimeClient
}
