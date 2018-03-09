/*
Copyright 2017 The Stolos Authors.
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
	"reflect"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/syncer/labeling"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/action"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

type namespaceActionBase struct {
	namespace string

	// Name of the operation being performed, mostly here for logging purposes.
	operation string

	// API Access related objects
	kubernetesInterface kubernetes.Interface
	namespaceLister     listers_core_v1.NamespaceLister
}

// NamespaceDeleteAction will delete a namespace when executed
type NamespaceDeleteAction struct {
	namespaceActionBase
}

// Resource implements Interface
func (s *namespaceActionBase) Resource() string {
	return "namespace"
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
	return fmt.Sprintf("%s.%s.%s", n.Resource(), n.Namespace(), n.Operation())
}

var _ action.Interface = &NamespaceDeleteAction{}

// NewNamespaceDeleteAction creates a new NamespaceDeleteAction for the given namespace
func NewNamespaceDeleteAction(
	namespace string,
	kubernetesInterface kubernetes.Interface,
	namespaceLister listers_core_v1.NamespaceLister) *NamespaceDeleteAction {
	return &NamespaceDeleteAction{
		namespaceActionBase: namespaceActionBase{
			namespace:           namespace,
			operation:           "delete",
			kubernetesInterface: kubernetesInterface,
			namespaceLister:     namespaceLister,
		},
	}
}

// Execute implements NamespaceAction
func (n *NamespaceDeleteAction) Execute() error {
	_, err := n.namespaceLister.Get(n.namespace)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Failed to get namespace %q from cache", n.namespace)
	}

	err = n.kubernetesInterface.CoreV1().Namespaces().Delete(n.namespace, &meta_v1.DeleteOptions{})
	if err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete namespace %q", n.namespace)
	}
	glog.Infof("Deleted namespace %s", n.namespace)
	return nil
}

// NamespaceUpsertAction will create or update a namespace when executed
type NamespaceUpsertAction struct {
	namespaceActionBase

	// Labels on the namespace.
	labels map[string]string
	// Owner reference for policy node
	ownerReferences []meta_v1.OwnerReference
}

var _ action.Interface = &NamespaceUpsertAction{}

// NewNamespaceUpsertAction creates a new NamespaceUpsertAction for the given namespace
func NewNamespaceUpsertAction(
	namespace string,
	uid types.UID,
	labels map[string]string,
	kubernetesInterface kubernetes.Interface,
	namespaceLister listers_core_v1.NamespaceLister) *NamespaceUpsertAction {
	blockOwnerDeletion := true
	return &NamespaceUpsertAction{
		namespaceActionBase: namespaceActionBase{
			namespace:           namespace,
			operation:           "upsert",
			kubernetesInterface: kubernetesInterface,
			namespaceLister:     namespaceLister,
		},
		labels: labeling.AddOriginLabelToMap(labels),
		ownerReferences: []meta_v1.OwnerReference{
			meta_v1.OwnerReference{
				APIVersion:         policyhierarchy_v1.SchemeGroupVersion.String(),
				Kind:               "PolicyNode",
				Name:               namespace,
				UID:                uid,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		},
	}
}

// Execute implements NamespaceAction
func (n *NamespaceUpsertAction) Execute() error {
	ns, err := n.namespaceLister.Get(n.namespace)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return n.create()
		}
		return errors.Wrapf(err, "Failed to get namespace %q during upsert", n.namespace)
	}

	return n.update(ns)
}

func (n *NamespaceUpsertAction) create() error {
	// Attempt to create namespace if it does not exist
	createdNamespace, err := n.kubernetesInterface.CoreV1().Namespaces().Create(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:            n.namespace,
			Labels:          n.labels,
			OwnerReferences: n.ownerReferences,
		},
	})

	if err != nil {
		if api_errors.IsAlreadyExists(err) {
			glog.Infof("Namespace %q already exists", n.namespace)
			return nil
		}
		return err
	}
	glog.Infof("Created namespace %q, resourceVersion %s", n.namespace, createdNamespace.ResourceVersion)
	return nil
}

func (n *NamespaceUpsertAction) update(currentNamespace *core_v1.Namespace) error {
	if reflect.DeepEqual(n.labels, currentNamespace.Labels) &&
		reflect.DeepEqual(n.ownerReferences, currentNamespace.OwnerReferences) {
		glog.Infof("Existing namespace %q does not need to be updated", n.namespace)
		return nil
	}

	glog.Infof("Updating namespace %q", n.namespace)
	updatedNamespace, err := n.kubernetesInterface.CoreV1().Namespaces().Update(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:            n.namespace,
			Labels:          n.labels,
			ResourceVersion: currentNamespace.ResourceVersion,
			OwnerReferences: n.ownerReferences,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to update namespace %q", n.namespace)
	}
	glog.Infof("Updated namespace %q, resourceVersion %s", n.namespace, updatedNamespace.ResourceVersion)
	return nil
}
