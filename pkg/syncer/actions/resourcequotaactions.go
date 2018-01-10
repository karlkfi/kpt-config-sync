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

// The actions needed by the syncer to perform operations on leaf (K8s native) Resource Quota objects
package actions

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/resource-quota"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

// ResourceQuotaAction represents a CRUD action on a resource quota spec
type ResourceQuotaAction interface {
	Interface
	// The resource quota spec
	ResourceQuotaSpec() core_v1.ResourceQuotaSpec
}

type resourceQuotaActionBase struct {
	// The resource quota spec to be created/updated by the action (upsert only)
	resourceQuotaSpec core_v1.ResourceQuotaSpec

	// The labels we are going to apply to the resource (upsert only)
	labels map[string]string

	// The namespace in which the resource quota object lives
	namespace string

	// The name of the operation being performed, mostly here for logging purposes.
	operation string

	// API Access related objects
	kubernetesInterface kubernetes.Interface
	resourceQuotaLister listers_core_v1.ResourceQuotaLister
}

// delete checks if the resource quota exists then deletes the object if it is not found in the cache.
func (n *resourceQuotaActionBase) delete() error {
	_, err := n.resourceQuotaLister.ResourceQuotas(n.namespace).Get(resource_quota.ResourceQuotaObjectName)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Failed to get resource quota from %s for delete", n.namespace)
	}
	return n.reallyDelete()
}

// reallyDelete performs the delete object on the store. This will ignore IsNotFound errors since
// that's the desired state of the store.
func (n *resourceQuotaActionBase) reallyDelete() error {
	err := n.kubernetesInterface.CoreV1().ResourceQuotas(n.namespace).Delete(
		resource_quota.ResourceQuotaObjectName, &meta_v1.DeleteOptions{})
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Failed to delete resource quota for %s", n.namespace)
	}
	glog.Infof("Deleted resource quota for namespace %s", n.namespace)
	return nil
}

// upsert will check if the resource exists then conditionally create or udpate it to the desired state.
func (n *resourceQuotaActionBase) upsert() error {
	resourceQuota, err := n.resourceQuotaLister.ResourceQuotas(n.namespace).Get(resource_quota.ResourceQuotaObjectName)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return n.reallyCreate()
		}
		return errors.Wrapf(err, "Failed to list resource quota object in %s for update", n.namespace)
	}

	return n.reallyUpdate(resourceQuota)
}

// updateResource will update an object's spec to match the spec set in the action.
func (n *resourceQuotaActionBase) reallyUpdate(current *core_v1.ResourceQuota) error {
	if reflect.DeepEqual(n.ResourceQuotaSpec, current.Spec) &&
		reflect.DeepEqual(current.Labels, n.labels) {
		return nil
	}

	createdResourceQuota, err := n.kubernetesInterface.CoreV1().ResourceQuotas(n.namespace).Update(
		&core_v1.ResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:            resource_quota.ResourceQuotaObjectName,
				Labels:          n.labels,
				ResourceVersion: current.ResourceVersion,
			},
			Spec:   n.resourceQuotaSpec,
			Status: current.Status,
		})
	if err != nil {
		return errors.Wrapf(err, "Failed to update quota for namespace %s", n.namespace)
	}
	glog.Infof("Updated resource quota for namespace %s, resourceVersion %s", n.namespace, createdResourceQuota.ResourceVersion)
	return nil
}

// reallyCreate performs the create operation on the store. If an exists error is returned, this
// will fall back to attempting an update.
func (n *resourceQuotaActionBase) reallyCreate() error {
	createdResourceQuota, err := n.kubernetesInterface.CoreV1().ResourceQuotas(n.namespace).Create(
		&core_v1.ResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   resource_quota.ResourceQuotaObjectName,
				Labels: n.labels,
			},
			Spec: n.resourceQuotaSpec,
		})
	if err != nil {
		if api_errors.IsAlreadyExists(err) {
			return n.upsert()
		}
		return errors.Wrapf(err, "Failed to create resource quota for %s", n.namespace)
	}
	glog.Infof("Created resource quota for namespace %s, resourceVersion %s", n.namespace, createdResourceQuota.ResourceVersion)
	return nil
}

func (n *resourceQuotaActionBase) ResourceQuotaSpec() core_v1.ResourceQuotaSpec {
	return n.resourceQuotaSpec
}

// Resource implements Interface
func (s *resourceQuotaActionBase) Resource() string {
	return "resourcequota"
}

func (n *resourceQuotaActionBase) Namespace() string {
	return n.namespace
}

func (n *resourceQuotaActionBase) Operation() string {
	return n.operation
}

// Resource implements Action
func (n *resourceQuotaActionBase) String() string {
	return fmt.Sprintf("%s.%s.%s", n.Resource(), n.Namespace(), n.Operation())
}

// ------- Delete -------
type ResourceQuotaDeleteAction struct {
	resourceQuotaActionBase
}

var _ ResourceQuotaAction = &ResourceQuotaDeleteAction{}

func NewResourceQuotaDeleteAction(
	namespace string,
	kubernetesInterface kubernetes.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister) *ResourceQuotaDeleteAction {
	return &ResourceQuotaDeleteAction{
		resourceQuotaActionBase: resourceQuotaActionBase{
			kubernetesInterface: kubernetesInterface,
			resourceQuotaLister: resourceQuotaLister,
			namespace:           namespace,
			operation:           "delete",
		},
	}
}

// Execute implements Interface
func (n *ResourceQuotaDeleteAction) Execute() error {
	return n.delete()
}

// ------- Upsert -------

// ResourceQuotaUpsertAction implements upserting to the backend.
type ResourceQuotaUpsertAction struct {
	resourceQuotaActionBase
}

// ResourceQuotaUpsertAction is a ResourceQuotaAction
var _ ResourceQuotaAction = &ResourceQuotaUpsertAction{}

// NewResourceQuotaUpsertAction creates an upsert action that will set
func NewResourceQuotaUpsertAction(
	namespace string,
	labels map[string]string,
	resourceQuotaSpec core_v1.ResourceQuotaSpec,
	kubernetesInterface kubernetes.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister,
) *ResourceQuotaUpsertAction {
	return &ResourceQuotaUpsertAction{
		resourceQuotaActionBase: resourceQuotaActionBase{
			kubernetesInterface: kubernetesInterface,
			resourceQuotaLister: resourceQuotaLister,
			resourceQuotaSpec:   resourceQuotaSpec,
			labels:              labels,
			namespace:           namespace,
			operation:           "upsert",
		},
	}
}

// Execute implements Interface
func (s *ResourceQuotaUpsertAction) Execute() error {
	return s.upsert()
}
