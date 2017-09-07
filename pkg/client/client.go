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
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client is a container for the kubernetes Clientset and adds some functionality on top of it for
// mostly reference purposes.
type Client struct {
	clientSet *kubernetes.Clientset
}

// ClusterState is the state of a cluster, which currently is just the list of namespaces.
type ClusterState struct {
	Namespaces []string
}

// GetState returns a ClusterState of the clusteer
func (c *Client) GetState() (*ClusterState, error) {
	namespaceList, err := c.clientSet.Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to list namespaces")
	}

	var namespaces []string
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	return &ClusterState{Namespaces: namespaces}, nil
}

// ClientSet returns the clientset in the client
func (c *Client) ClientSet() *kubernetes.Clientset {
	return c.clientSet
}
