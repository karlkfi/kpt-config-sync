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
	"regexp"
	"strings"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var reservedNamespaces = map[string]bool{
	"default":     true,
	"kube-public": true,
	"kube-system": true,
}

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

// ClusterState is the state of a cluster, which currently is just the list of namespaces.
type ClusterState struct {
	Namespaces []string
}

// GetState returns a ClusterState of the clusteer
func (c *Client) GetState() (*ClusterState, error) {
	namespaceList, err := c.kubernetesClientset.CoreV1().Namespaces().List(meta_v1.ListOptions{})
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
func (c *Client) Kubernetes() *kubernetes.Clientset {
	return c.kubernetesClientset
}

// PolicyHierarchy returns the clientset for the policyhierarchy custom resource
func (c *Client) PolicyHierarchy() *policyhierarchy.Clientset {
	return c.policyHierarchyClientset
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
			namespaceActions = append(namespaceActions, c.NamespaceDeleteAction(ns))
		} else {
			glog.Infof("Namespace %s exists, no change needed", ns)
		}
	}

	for ns := range definedNamespaces {
		if !existingNamespaces[ns] {
			namespaceActions = append(namespaceActions, c.NamespaceCreateAction(ns))
		}
	}

	for _, action := range namespaceActions {
		err := action.Execute()
		if err != nil {
			glog.Infof("Action %s %s failed due to %s", action.Name(), action.Operation(), err)
		}
	}

	return nil
}

func (c *Client) FetchPolicyHierarchy() ([]policyhierarchy_v1.PolicyNode, string, error) {
	glog.Info("Fetching policy hierarchy")

	nodeList, err := c.policyHierarchyClientset.K8usV1().PolicyNodes().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, "", errors.Wrapf(err, "Failed to list policy hierarchy")
	}

	return nodeList.Items, nodeList.ResourceVersion, nil
}

// ExtractNamespaces returns all the namespace names from a list of policy nodes.
func ExtractNamespaces(policyNodes []policyhierarchy_v1.PolicyNode) []string {
	var namespaces []string
	for _, policyNode := range policyNodes {
		namespaces = append(namespaces, ExtractNamespace(&policyNode))
	}

	return namespaces
}

// Pattern from output returned by kubectl
var namespaceRegexPattern = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
var namespaceRe *regexp.Regexp

func init() {
	// Have to declare err here because golang won't let me use := on the next line.  Behold the perfect beauty of go.
	var err error
	namespaceRe, err = regexp.Compile(namespaceRegexPattern)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to compile regex"))
	}
}

// ExtractNamespace returns the sanitized namespace name from a PolicyNode
func ExtractNamespace(policyNode *policyhierarchy_v1.PolicyNode) string {
	// TODO: add check for invalid characters
	ns := policyNode.Spec.Name
	if !namespaceRe.MatchString(ns) {
		panic(errors.Errorf("Namespace %s does not satisfy valid namespace patter %s", ns, namespaceRegexPattern))
	}
	return strings.ToLower(ns)
}

func (c *Client) SyncPolicyHierarchy() error {
	policyNodes, _, err := c.FetchPolicyHierarchy()
	if err != nil {
		return err
	}
	nss := ExtractNamespaces(policyNodes)
	return c.SyncNamespaces(nss)
}

func (c *Client) NamespaceCreateAction(namespace string) *NamespaceCreateAction {
	return &NamespaceCreateAction{
		namespaceActionBase{
			namespace:     namespace,
			clusterClient: c,
		},
	}
}

func (c *Client) NamespaceDeleteAction(namespace string) *NamespaceDeleteAction {
	return &NamespaceDeleteAction{
		namespaceActionBase{
			namespace:     namespace,
			clusterClient: c,
		},
	}
}
