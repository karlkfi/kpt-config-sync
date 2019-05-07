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

// Package meta sets up a set of client sets that we use for communicating with core Kubernetes
// as well as the custom resources.
package meta

import (
	"time"

	"github.com/google/nomos/clientgen/apis"
	"github.com/pkg/errors"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Interface specifies the interface for Client.
type Interface interface {
	Kubernetes() kubernetes.Interface
	ConfigManagement() apis.Interface
	APIExtensions() apiextensions.Interface
	Runtime() client.Client
}

// Client is a container for the kubernetes Clientset and the configmanagement clientset.
type Client struct {
	kubernetesClientset       *kubernetes.Clientset
	configManagementClientset *apis.Clientset
	apiExtensionsClientset    *apiextensions.Clientset
	runtimeClient             client.Client
}

var _ Interface = &Client{}

// Kubernetes returns the kubernetes clientset
func (c *Client) Kubernetes() kubernetes.Interface {
	return c.kubernetesClientset
}

// ConfigManagement returns the configmanagement clientset
func (c *Client) ConfigManagement() apis.Interface {
	return c.configManagementClientset
}

// APIExtensions returns the ApiExtensions clientset
func (c *Client) APIExtensions() apiextensions.Interface {
	return c.apiExtensionsClientset
}

// Runtime returns the kubernetes runtime client for CRUD operations.
func (c *Client) Runtime() client.Client {
	return c.runtimeClient
}

// New creates a new Client directly from member client sets.
func New(
	kubernetesClientset *kubernetes.Clientset,
	configManagementClientset *apis.Clientset,
	apiExtensionsClientset *apiextensions.Clientset,
	runtimeClient client.Client) *Client {
	return &Client{
		kubernetesClientset:       kubernetesClientset,
		configManagementClientset: configManagementClientset,
		apiExtensionsClientset:    apiExtensionsClientset,
		runtimeClient:             runtimeClient,
	}
}

// NewForConfig will r
func NewForConfig(cfg *rest.Config, syncPeriod *time.Duration) (*Client, error) {
	kubernetesClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create kubernetes clientset")
	}

	configManagementClientset, err := apis.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create configmanagement clientset")
	}

	apiExtensionsClientset, err := apiextensions.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create apiextensions clientset")
	}

	mgr, err := manager.New(cfg, manager.Options{SyncPeriod: syncPeriod})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create manager")
	}

	return New(kubernetesClientset, configManagementClientset, apiExtensionsClientset, mgr.GetClient()), nil
}

// NewForConfigOrDie creates a new Client from the given config and panics if there is an error.
func NewForConfigOrDie(cfg *rest.Config) *Client {
	return &Client{
		kubernetesClientset:       kubernetes.NewForConfigOrDie(cfg),
		configManagementClientset: apis.NewForConfigOrDie(cfg),
		apiExtensionsClientset:    apiextensions.NewForConfigOrDie(cfg),
	}
}
