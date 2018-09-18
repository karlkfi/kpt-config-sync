/*
Copyright 2018 The Nomos Authors.
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

package modules

import (
	"reflect"

	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	controllerinformers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// ClusterRoles implements a module for comparing clusterroles and
// generating actions to update them.
type ClusterRoles struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ clusterpolicycontroller.Module = &ClusterRoles{}

// NewClusterRoles creates the module.
func NewClusterRoles(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *ClusterRoles {
	return &ClusterRoles{
		client:    client,
		informers: informers,
	}
}

// Name implements clusterpolicycontroller.Module.
func (s *ClusterRoles) Name() string {
	return "ClusterRoles"
}

// Equal implements clusterpolicycontroller.Module.
func (s *ClusterRoles) Equal(lhsObj metav1.Object, rhsObj metav1.Object) bool {
	lhs := lhsObj.(*rbacv1.ClusterRole)
	rhs := rhsObj.(*rbacv1.ClusterRole)

	if lhs == nil || rhs == nil {
		return lhs == rhs
	}

	if lhs.AggregationRule != nil || rhs.AggregationRule != nil {
		return reflect.DeepEqual(lhs.AggregationRule, rhs.AggregationRule)
	}
	return reflect.DeepEqual(lhs.Rules, rhs.Rules)
}

// equalSpec performs equals on runtime.Objects
func (s *ClusterRoles) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(metav1.Object), rhsObj.(metav1.Object))
}

// InformerProvider implements clusterpolicycontroller.Module
func (s *ClusterRoles) InformerProvider() controllerinformers.InformerProvider {
	return s.informers.Rbac().V1().ClusterRoles()
}

// Instance implements clusterpolicycontroller.Module
func (s *ClusterRoles) Instance() metav1.Object {
	return &rbacv1.ClusterRole{}
}

// Extract implements clusterpolicycontroller.Module
func (s *ClusterRoles) Extract(clusterPolicy *policyhierarchyv1.ClusterPolicy) []metav1.Object {
	var roles []runtime.Object
	for _, r := range clusterPolicy.Spec.ClusterRolesV1 {
		roles = append(roles, r.DeepCopy())
	}
	return object.RuntimeToMeta(roles)
}

// ActionSpec implements clusterpolicycontroller.Module
func (s *ClusterRoles) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&rbacv1.ClusterRole{},
		corev1.SchemeGroupVersion,
		s.equalSpec,
		s.client.RbacV1(),
		s.informers.Rbac().V1().ClusterRoles().Lister(),
	)
}
