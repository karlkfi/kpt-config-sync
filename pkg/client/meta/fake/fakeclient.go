/*
Copyright 2017 The Kubernetes Authors.

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
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/meta"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	fakepolicyhierarchy "github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy/fake"
	"k8s.io/client-go/kubernetes"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
)

// Client implements meta.Interface with fake clientsets.
type Client struct {
	KubernetesClientset      *fakekubernetes.Clientset
	PolicyhierarchyClientset *fakepolicyhierarchy.Clientset
}

var _ meta.Interface = &Client{}

// NewClient creates a FakeClient with default simple clientsets
func NewClient() *Client {
	return &Client{
		KubernetesClientset:      fakekubernetes.NewSimpleClientset(),
		PolicyhierarchyClientset: fakepolicyhierarchy.NewSimpleClientset(),
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
