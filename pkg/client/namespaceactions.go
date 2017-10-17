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
package client

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// TODO: Move this file out of client package

// NamespaceAction represents a CRUD action on a namespace
type NamespaceAction interface {
	// Operation returns the operation name
	Operation() string
	// Execute will execute the operation then return an error on failure
	Execute() error
	// Name returns the name of the namespace being operated on
	Name() string
}

type namespaceActionBase struct {
	namespace           string
	kubernetesInterface kubernetes.Interface
}

// NamespaceDeleteAction will delete a namespace when executed
type NamespaceDeleteAction struct {
	namespaceActionBase
}

// Name implements NamespaceAction
func (n *namespaceActionBase) Name() string {
	return n.namespace
}

var _ NamespaceAction = &NamespaceDeleteAction{}

// NewNamespaceDeleteAction creates a new NamespaceDeleteAction for the given namespace
func NewNamespaceDeleteAction(kubernetesInterface kubernetes.Interface, namespace string) *NamespaceDeleteAction {
	return &NamespaceDeleteAction{
		namespaceActionBase: namespaceActionBase{
			kubernetesInterface: kubernetesInterface,
			namespace:           namespace,
		},
	}
}

// Operation implements NamespaceAction
func (n *NamespaceDeleteAction) Operation() string {
	return "delete"
}

// Execute implements NamespaceAction
func (n *NamespaceDeleteAction) Execute() error {
	glog.Infof("Deleting namespace %s", n.namespace)
	return n.kubernetesInterface.CoreV1().Namespaces().Delete(n.namespace, &meta_v1.DeleteOptions{})
}

// NamespaceCreateAction will create a namespace when executed
type NamespaceCreateAction struct {
	namespaceActionBase
}

var _ NamespaceAction = &NamespaceCreateAction{}

// NewNamespaceCreateAction creates a new NamespaceCreateAction for the given namespace
func NewNamespaceCreateAction(kubernetesInterface kubernetes.Interface, namespace string) *NamespaceCreateAction {
	return &NamespaceCreateAction{
		namespaceActionBase: namespaceActionBase{
			kubernetesInterface: kubernetesInterface,
			namespace:           namespace,
		},
	}
}

// Operation implements NamespaceAction
func (n *NamespaceCreateAction) Operation() string {
	return "create"
}

// waitForTerminatingNamespace waits for a namespace to go from terminating to fully deleted.
func (n *NamespaceCreateAction) waitForTerminatingNamespace(resourceVersion string) error {
	watchInterface, err := n.kubernetesInterface.CoreV1().Namespaces().Watch(meta_v1.ListOptions{
		ResourceVersion: resourceVersion,
		FieldSelector:   fmt.Sprintf("metadata.name=%s", n.namespace),
	})
	for event := range watchInterface.ResultChan() {
		glog.V(1).Infof("Namesapce %s %s event", n.namespace, event.Type)
		switch event.Type {
		case watch.Added:
			return errors.Errorf(
				"Consistency error, namespace %s was terminating bug got create event: %#v", n.namespace, event.Object)
		case watch.Modified:
			glog.V(7).Infof("Namespace %s modified, still waiting", n.namespace)
		case watch.Deleted:
			glog.V(7).Infof("Namespace %s deleted, done waiting", n.namespace)
			watchInterface.Stop()
			return nil
		case watch.Error:
			return errors.Wrapf(
				err, "Error while waiting for namespace %s to terminate for create", n.namespace)
		}
	}
	return errors.Errorf("Namespace %s was not deleted before event watch shut down", n.namespace)
}

// handleTerminatingNamespace will check if the namespace exists as a terminating namespace
// and then wait for termination to complete before returning.
func (n *NamespaceCreateAction) handleTerminatingNamespace() error {
	// Check if the namespace is terminating.
	existingNamespace, err := n.kubernetesInterface.CoreV1().Namespaces().Get(n.namespace, meta_v1.GetOptions{})
	if err != nil {
		if api_errors.IsNotFound(err) {
			glog.V(7).Infof("Namespace %s does not exist, nothing to wait for", n.namespace)
			return nil
		}
		return errors.Wrapf(err, "Failed to check for existing namespace")
	}

	if existingNamespace.Status.Phase != core_v1.NamespaceTerminating {
		return errors.Errorf(
			"Consistency error, namespace %s in unexpected state %s: %#v",
			n.namespace, existingNamespace.Status.Phase, existingNamespace)
	}

	glog.V(7).Infof("Namespace %s is terminating, will wait until deleted", n.namespace)
	return n.waitForTerminatingNamespace(existingNamespace.ResourceVersion)
}

// Execute implements NamespaceAction
func (n *NamespaceCreateAction) Execute() error {
	err := n.handleTerminatingNamespace()
	if err != nil {
		return err
	}

	glog.Infof("Creating namespace %s", n.namespace)
	createdNamespace, err := n.kubernetesInterface.CoreV1().Namespaces().Create(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: n.namespace,
		},
	})
	if err != nil {
		glog.Infof("Failed to create namespace %s: %v", n.namespace, err)
		return err
	}
	glog.Infof("Created namespace %s, resourceVersion %s", n.namespace, createdNamespace.ResourceVersion)
	return nil
}
