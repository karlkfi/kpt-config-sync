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

// Package meta sets up a set of client sets that we use for communicating with core Kubernetes
// as well as the custom resources.
package meta

import (
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Interface specifies the interface for Client.
type Interface interface {
	Kubernetes() kubernetes.Interface
	PolicyHierarchy() policyhierarchy.Interface
}

// Client is a container for the kubernetes Clientset and the policyhierarchy clientset.
type Client struct {
	kubernetesClientset      *kubernetes.Clientset
	policyHierarchyClientset *policyhierarchy.Clientset
}

var _ Interface = &Client{}

// Kubernetes returns the kubernetes clientset
func (c *Client) Kubernetes() kubernetes.Interface {
	return c.kubernetesClientset
}

// PolicyHierarchy returns the policyhierarchy clientse
func (c *Client) PolicyHierarchy() policyhierarchy.Interface {
	return c.policyHierarchyClientset
}

// New creates a new Client directly from member client sets.
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

// NewForConfigOrDie creates a new Client from the given config and panics if there is an error.
func NewForConfigOrDie(cfg *rest.Config) *Client {
	return &Client{
		kubernetesClientset:      kubernetes.NewForConfigOrDie(cfg),
		policyHierarchyClientset: policyhierarchy.NewForConfigOrDie(cfg),
	}
}
