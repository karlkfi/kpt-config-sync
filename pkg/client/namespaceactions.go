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
	"flag"

	"github.com/golang/glog"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var dryRun = flag.Bool(
	"dry_run", false, "Don't perform actions, just log what would have happened")

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

func (n *NamespaceDeleteAction) Operation() string {
	return "delete"
}

func (n *NamespaceDeleteAction) Execute() error {
	if *dryRun {
		glog.Infof("Would have deleted namespace %s", n.namespace)
		return nil
	}

	glog.Infof("Deleting namespace %s", n.namespace)
	return n.kubernetesInterface.CoreV1().Namespaces().Delete(n.namespace, &meta_v1.DeleteOptions{})
}

func (n *NamespaceDeleteAction) Name() string {
	return n.namespace
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

func (n *NamespaceCreateAction) Operation() string {
	return "create"
}

func (n *NamespaceCreateAction) Execute() error {
	if *dryRun {
		glog.Infof("Would have created namespace %s", n.namespace)
		return nil
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

func (n *NamespaceCreateAction) Name() string {
	return n.namespace
}
