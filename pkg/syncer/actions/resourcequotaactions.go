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

// The actions needed by the syncer to perform operations on leaf (K8s native) Resource Quota objects
package actions

import (
	"github.com/golang/glog"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/google/stolos/pkg/resource-quota"
	"fmt"
)

// ResourceQuotaAction represents a CRUD action on a resource quota spec
type ResourceQuotaAction interface {
	Interface
	// The resource quota spec
	ResourceQuotaSpec() core_v1.ResourceQuotaSpec
}

type resourceQuotaActionBase struct {
	// The resource quota spec to be created/updated by the action
	resourceQuotaSpec core_v1.ResourceQuotaSpec
	// The namespace in which the resource quota object lives
	namespace string
	operation string
	// The kubernetes interface
	kubernetesInterface kubernetes.Interface
}

func (n *resourceQuotaActionBase) ResourceQuotaSpec() core_v1.ResourceQuotaSpec {
	return n.resourceQuotaSpec
}

func (n *resourceQuotaActionBase) Namespace() string {
	return n.namespace
}

func (n *resourceQuotaActionBase) Operation() string {
	return n.operation
}

// Resource implements Action
func (n *resourceQuotaActionBase) String() string {
	return fmt.Sprintf("resourcequota.%s.%s", n.Namespace(), n.Operation())
}

// ------- Delete -------
type ResourceQuotaDeleteAction struct {
	resourceQuotaActionBase
}

var _ ResourceQuotaAction = &ResourceQuotaDeleteAction{}

func NewResourceQuotaDeleteAction(kubernetesInterface kubernetes.Interface, namespace string) *ResourceQuotaDeleteAction {
	return &ResourceQuotaDeleteAction{
		resourceQuotaActionBase: resourceQuotaActionBase{
			kubernetesInterface: kubernetesInterface,
			namespace:           namespace,
			operation:           "delete",
		},
	}
}

func (n *ResourceQuotaDeleteAction) Execute() error {
	err := n.kubernetesInterface.CoreV1().ResourceQuotas(n.namespace).Delete(resource_quota.ResourceQuotaObjectName, &meta_v1.DeleteOptions{})
	if err != nil {
		glog.Infof("Failed to delete resource quota for namespace %s: %v", n.namespace, err)
		return err
	}
	glog.Infof("Deleted resource quota for namespace %s", n.namespace)
	return nil
}

// ------- Create -------
type ResourceQuotaCreateAction struct {
	resourceQuotaActionBase
}

var _ ResourceQuotaAction = &ResourceQuotaCreateAction{}

func NewResourceQuotaCreateAction(kubernetesInterface kubernetes.Interface, namespace string,
	resourceQuotaSpec core_v1.ResourceQuotaSpec) *ResourceQuotaCreateAction {
	return &ResourceQuotaCreateAction{
		resourceQuotaActionBase: resourceQuotaActionBase{
			kubernetesInterface: kubernetesInterface,
			resourceQuotaSpec:   resourceQuotaSpec,
			namespace:           namespace,
			operation:           "create",
		},
	}
}

func (n *ResourceQuotaCreateAction) Execute() error {
	createdResourceQuota, err := n.kubernetesInterface.CoreV1().ResourceQuotas(n.namespace).Create(&core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   resource_quota.ResourceQuotaObjectName,
			Labels: resource_quota.StolosQuotaLabels,
		},
		Spec: n.resourceQuotaSpec,
	})
	if err != nil {
		glog.Infof("Failed to create resource quota for namespace %s: %v", n.namespace, err)
		return err
	}
	glog.Infof("Created resource quota for namespace %s, resourceVersion %s", n.namespace, createdResourceQuota.ResourceVersion)
	return nil
}

// ------- Update -------
type ResourceQuotaUpdateAction struct {
	resourceQuotaActionBase
	// The resource version of the last known state resource quota, to ensure updates are atomic.
	resourceVersion string
}

var _ ResourceQuotaAction = &ResourceQuotaUpdateAction{}

func NewResourceQuotaUpdateAction(kubernetesInterface kubernetes.Interface, namespace string,
	resourceQuotaSpec core_v1.ResourceQuotaSpec, resourceVersion string) *ResourceQuotaUpdateAction {
	return &ResourceQuotaUpdateAction{
		resourceQuotaActionBase: resourceQuotaActionBase{
			kubernetesInterface: kubernetesInterface,
			resourceQuotaSpec:   resourceQuotaSpec,
			namespace:           namespace,
			operation:           "update",
		},
		resourceVersion: resourceVersion,
	}
}

func (n *ResourceQuotaUpdateAction) Execute() error {
	createdResourceQuota, err := n.kubernetesInterface.CoreV1().ResourceQuotas(n.namespace).Update(&core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{Name: resource_quota.ResourceQuotaObjectName, ResourceVersion: n.resourceVersion},
		Spec:       n.resourceQuotaSpec,
	})
	if err != nil {
		glog.Infof("Failed to update resource quota for namespace %s: %v", n.namespace, err)
		return err
	}
	glog.Infof("Updated resource quota for namespace %s, resourceVersion %s", n.namespace, createdResourceQuota.ResourceVersion)
	return nil
}
