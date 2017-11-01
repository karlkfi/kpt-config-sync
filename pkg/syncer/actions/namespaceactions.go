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
package actions

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

type namespaceActionBase struct {
	namespace           string
	operation           string
	kubernetesInterface kubernetes.Interface
	namespaceLister     listers_core_v1.NamespaceLister
}

// NamespaceDeleteAction will delete a namespace when executed
type NamespaceDeleteAction struct {
	namespaceActionBase
}

// Name implements Action
func (n *namespaceActionBase) Namespace() string {
	return n.namespace
}

// Name implements Action
func (n *namespaceActionBase) Operation() string {
	return n.operation
}

// String implements Action
func (n *namespaceActionBase) String() string {
	return fmt.Sprintf("namespace.%s.%s", n.Namespace(), n.Operation())
}

var _ Interface = &NamespaceDeleteAction{}

// NewNamespaceDeleteAction creates a new NamespaceDeleteAction for the given namespace
func NewNamespaceDeleteAction(
	kubernetesInterface kubernetes.Interface,
	namespace string,
	namespaceLister listers_core_v1.NamespaceLister) *NamespaceDeleteAction {
	return &NamespaceDeleteAction{
		namespaceActionBase: namespaceActionBase{
			kubernetesInterface: kubernetesInterface,
			namespace:           namespace,
			operation:           "delete",
			namespaceLister:     namespaceLister,
		},
	}
}

// Execute implements NamespaceAction
func (n *NamespaceDeleteAction) Execute() error {
	glog.Infof("Deleting namespace %s", n.namespace)
	_, err := n.namespaceLister.Get(n.namespace)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Failed to get namespace %s from cache")
	}

	err = n.kubernetesInterface.CoreV1().Namespaces().Delete(n.namespace, &meta_v1.DeleteOptions{})
	if err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete namespace %s", n.namespace)
	}
	return nil
}

// NamespaceCreateAction will create a namespace when executed
type NamespaceCreateAction struct {
	namespaceActionBase
}

var _ Interface = &NamespaceCreateAction{}

// NewNamespaceCreateAction creates a new NamespaceCreateAction for the given namespace
func NewNamespaceCreateAction(
	kubernetesInterface kubernetes.Interface,
	namespace string,
	namespaceLister listers_core_v1.NamespaceLister) *NamespaceCreateAction {
	return &NamespaceCreateAction{
		namespaceActionBase: namespaceActionBase{
			kubernetesInterface: kubernetesInterface,
			namespace:           namespace,
			operation:           "create",
			namespaceLister:     namespaceLister,
		},
	}
}

// Execute implements NamespaceAction
func (n *NamespaceCreateAction) Execute() error {
	ns, err := n.namespaceLister.Get(n.namespace)
	if err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to get namespace %s during create", n.namespace)
	}
	if err == nil && ns.Status.Phase == core_v1.NamespaceActive {
		return nil
	}

	// Attempt to create namespace if it does not exist, or exists and is terminating.
	glog.Infof("Creating namespace %s", n.namespace)
	createdNamespace, err := n.kubernetesInterface.CoreV1().Namespaces().Create(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: n.namespace,
		},
	})

	if err != nil {
		if api_errors.IsAlreadyExists(err) {
			glog.Infof("Namespace %s already exists in phase %s", createdNamespace.Status.Phase)
			return nil
		}
		glog.Infof("Failed to create namespace %s: %v", n.namespace, err)
		return err
	}
	glog.Infof("Created namespace %s, resourceVersion %s", n.namespace, createdNamespace.ResourceVersion)
	return nil
}
