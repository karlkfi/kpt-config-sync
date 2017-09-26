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

// A kubernetes API server client helper.
package client

import (
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client is a container for the kubernetes Clientset and adds some functionality on top of it for
// mostly reference purposes.
type Client struct {
	kubernetesClientset      *kubernetes.Clientset
	policyHierarchyClientset *policyhierarchy.Clientset
}

// New creates a new Client from the clientsets it will use.
func New(
	kubernetesClientset *kubernetes.Clientset,
	policyHierarchyClientset *policyhierarchy.Clientset) *Client {
	return &Client{
		kubernetesClientset:      kubernetesClientset,
		policyHierarchyClientset: policyHierarchyClientset,
	}
}

// NewForConfig will r
func NewForConfig(cfg *rest.Config) (*Client, error) {
	kubernetesClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create kubernetes clientset")
	}

	policyHierarchyClientSet, err := policyhierarchy.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create policyhierarchy clientset")
	}

	return New(kubernetesClientset, policyHierarchyClientSet), nil
}

// ClientSet returns the clientset in the client
func (c *Client) Kubernetes() *kubernetes.Clientset {
	return c.kubernetesClientset
}

// PolicyHierarchy returns the clientset for the policyhierarchy custom resource
func (c *Client) PolicyHierarchy() *policyhierarchy.Clientset {
	return c.policyHierarchyClientset
}
