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
	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/service"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/util/namespaceutil"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
		if namespaceutil.IsReserved(ns) {
			continue
		}
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

// FetchPolicyHierarchy returns all policy nodes from the custom resource as well as
// the resource version they were read at.
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

// ExtractNamespace returns the sanitized namespace name from a PolicyNode
func ExtractNamespace(policyNode *policyhierarchy_v1.PolicyNode) string {
	return namespaceutil.SanitizeNamespace(policyNode.Spec.Name)
}

// WrapPolicyNodeSpec will take a PolicyNodeSpec, wrap it in a PolicyNode and populate the appropriate
// fields.
func WrapPolicyNodeSpec(spec *policyhierarchy_v1.PolicyNodeSpec) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		TypeMeta: meta_v1.TypeMeta{
			APIVersion: policyhierarchy_v1.GroupName,
			Kind:       "PolicyNode",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceutil.SanitizeNamespace(spec.Name),
		},
		Spec: *spec,
	}
}

// SyncPolicyHierarchy will read PolicyNodes from the custom resource then synchronize the
// namespaces to match the ones defined in the PolicyNodes custom resource.
func (c *Client) SyncPolicyHierarchy() error {
	policyNodes, _, err := c.FetchPolicyHierarchy()
	if err != nil {
		return err
	}
	nss := ExtractNamespaces(policyNodes)
	return c.SyncNamespaces(nss)
}

// NamespaceCreateAction will return a NamespaceAction that will create a namespace.
func (c *Client) NamespaceCreateAction(namespace string) *NamespaceCreateAction {
	return NewNamespaceCreateAction(c.kubernetesClientset, namespace)
}

// NamespaceDeleteAction will return a NamespaceAction that will delete a namespace
func (c *Client) NamespaceDeleteAction(namespace string) *NamespaceDeleteAction {
	return NewNamespaceDeleteAction(c.kubernetesClientset, namespace)
}

// RunSyncerDaemon will read the policynodes custom resource then proceed to synchronize the namespaces in the cluster
// once synced, it will watch the policynodes resource for changes and incrementally apply those changes.
// Note that this may need to be modified to deal with the hysterisis between deleting a namespace and having it fully
// removed from k8s since the operation is not synchronous and a delete-create-delete may cause issues.
// Returns a callback to stop the daemon.
func (c *Client) RunSyncerDaemon() service.Stoppable {
	policyNodes, resourceVersion, err := c.FetchPolicyHierarchy()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to fetch policies"))
	}

	namespaces := ExtractNamespaces(policyNodes)

	err = c.SyncNamespaces(namespaces)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to sync namespaces"))
	}

	watchIface, err := c.PolicyHierarchy().K8usV1().PolicyNodes().Watch(
		meta_v1.ListOptions{ResourceVersion: resourceVersion})
	if err != nil {
		panic(errors.Wrapf(err, "Failed to watch policy hierarchy"))
	}

	// TODO: refactor this into wrapper which re-opens watch on resultChan closing
	go func() {
		glog.Infof("Watching changes to policynodes at %s", resourceVersion)
		resultChan := watchIface.ResultChan()
		for {
			select {
			case event, ok := <-resultChan:
				if !ok {
					glog.Info("Channel closed, exiting")
					return
				}
				node := event.Object.(*policyhierarchy_v1.PolicyNode)
				glog.Infof("Got event %s %s resourceVersion %s", event.Type, node.Spec.Name, node.ResourceVersion)

				namespace := ExtractNamespace(node)

				var action NamespaceAction
				switch event.Type {
				case watch.Added:
					// add the ns
					action = c.NamespaceCreateAction(namespace)
				case watch.Modified:
				case watch.Deleted:
					// delete the ns
					action = c.NamespaceDeleteAction(namespace)
				case watch.Error:
					panic(errors.Wrapf(err, "Got error during watch operation"))
				}

				if action == nil {
					continue
				}

				err := action.Execute()
				if err != nil {
					glog.Errorf("Failed to perform action %s on %s: %s", action.Operation(), action.Name(), err)
				}
			}
		}
	}()

	return watchIface
}
