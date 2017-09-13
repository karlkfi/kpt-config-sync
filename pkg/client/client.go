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
	"github.com/golang/glog"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var reservedNamespaces = map[string]bool{
	"default":     true,
	"kube-public": true,
	"kube-system": true,
}

// Client is a container for the kubernetes Clientset and adds some functionality on top of it for
// mostly reference purposes.
type Client struct {
	clientSet                *kubernetes.Clientset
	policyHierarchyClientset *policyhierarchy.Clientset
}

// ClusterState is the state of a cluster, which currently is just the list of namespaces.
type ClusterState struct {
	Namespaces []string
}

// GetState returns a ClusterState of the clusteer
func (c *Client) GetState() (*ClusterState, error) {
	namespaceList, err := c.clientSet.CoreV1().Namespaces().List(meta_v1.ListOptions{})
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

// SyncNamespaces updates / deletes namespaces on the cluster to match the list of namespaces
// provided in the arg.  This ignores kube-public, kube-system and default.
func (c *Client) SyncNamespaces(namespaces []string) error {
	glog.Infof("Syncing %d namespaces", len(namespaces))

	clusterState, err := c.GetState()
	if err != nil {
		return errors.Wrapf(err, "Failed to get state from cluster")
	}

	definedNamespaces := map[string]bool{}
	for _, namespace := range namespaces {
		definedNamespaces[namespace] = true
	}
	existingNamespaces := map[string]bool{}
	for _, namespace := range clusterState.Namespaces {
		existingNamespaces[namespace] = true
	}

	namespaceActions := []NamespaceAction{}
	for ns := range existingNamespaces {
		if reservedNamespaces[ns] {
			continue
		}
		if !definedNamespaces[ns] {
			namespaceActions = append(namespaceActions, &NamespaceDeleteAction{namespace: ns})
		} else {
			glog.Infof("Namespace %s exists, no change needed", ns)
		}
	}

	for ns := range definedNamespaces {
		if !existingNamespaces[ns] {
			namespaceActions = append(namespaceActions, &NamespaceCreateAction{namespace: ns})
		}
	}

	for _, action := range namespaceActions {
		action.Execute(c)
	}

	return nil
}
