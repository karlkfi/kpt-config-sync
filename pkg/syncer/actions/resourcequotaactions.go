/*
Copyright 2017 The Nomos Authors.
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
// Reviewed by sunilarora

package actions

import (
	"reflect"

	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/resourcequota"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

// NewResourceQuotaDeleteAction creates a delete action that will remove quota limits.
func NewResourceQuotaDeleteAction(
	namespace string,
	kubernetesInterface kubernetes.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister) *action.ReflectiveDeleteAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(core_v1.ResourceQuota{}),
		KindPlural: action.Plural(core_v1.ResourceQuota{}),
		Group:      core_v1.SchemeGroupVersion.Group,
		Version:    core_v1.SchemeGroupVersion.Version,
		EqualSpec:  ResourceQuotasEqual,
		Client:     kubernetesInterface.CoreV1(),
		Lister:     resourceQuotaLister,
	}
	return action.NewReflectiveDeleteAction(namespace, resourcequota.ResourceQuotaObjectName, spec)
}

// NewResourceQuotaUpsertAction creates an upsert action that will create/update quota limits.
func NewResourceQuotaUpsertAction(
	namespace string,
	labels map[string]string,
	resourceQuotaSpec core_v1.ResourceQuotaSpec,
	kubernetesInterface kubernetes.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister,
) *action.ReflectiveUpsertAction {
	spec := &action.ReflectiveActionSpec{
		Resource:   action.LowerPlural(core_v1.ResourceQuota{}),
		KindPlural: action.Plural(core_v1.ResourceQuota{}),
		Group:      core_v1.SchemeGroupVersion.Group,
		Version:    core_v1.SchemeGroupVersion.Version,
		EqualSpec:  ResourceQuotasEqual,
		Client:     kubernetesInterface.CoreV1(),
		Lister:     resourceQuotaLister,
	}
	quota := &core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:   resourcequota.ResourceQuotaObjectName,
			Labels: labels,
		},
		Spec: resourceQuotaSpec,
	}
	return action.NewReflectiveUpsertAction(namespace, resourcequota.ResourceQuotaObjectName, quota, spec)
}

// ResourceQuotasEqual returns true if both resource quotas have functionally equivalent limits.
func ResourceQuotasEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lQuota := lhs.(*core_v1.ResourceQuota)
	rQuota := rhs.(*core_v1.ResourceQuota)
	return reflect.DeepEqual(lQuota.Spec, rQuota.Spec) && lQuota.Name == rQuota.Name
}
