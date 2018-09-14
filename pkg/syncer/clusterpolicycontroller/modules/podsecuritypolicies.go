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

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/object"
	"github.com/google/nomos/pkg/syncer/clusterpolicycontroller"
	controller_informers "github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	v1beta "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// PodSecurityPolicies implements a module for comparing
// podsecuritypolicies and generating actions to update them.
type PodSecurityPolicies struct {
	client    kubernetes.Interface
	informers informers.SharedInformerFactory
}

var _ clusterpolicycontroller.Module = &PodSecurityPolicies{}

// NewPodSecurityPolicies creates the module.
func NewPodSecurityPolicies(
	client kubernetes.Interface, informers informers.SharedInformerFactory) *PodSecurityPolicies {
	return &PodSecurityPolicies{
		client:    client,
		informers: informers,
	}
}

// Name implements clusterpolicycontroller.Module.
func (s *PodSecurityPolicies) Name() string {
	return "PodSecurityPolicies"
}

// Equal implements clusterpolicycontroller.Module.
func (s *PodSecurityPolicies) Equal(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*v1beta.PodSecurityPolicy)
	rhs := rhsObj.(*v1beta.PodSecurityPolicy)

	if lhs == nil || rhs == nil {
		return lhs == rhs
	}

	return reflect.DeepEqual(lhs.Spec, rhs.Spec)
}

// equalSpec performs equals on runtime.Objects
func (s *PodSecurityPolicies) equalSpec(lhsObj runtime.Object, rhsObj runtime.Object) bool {
	return s.Equal(lhsObj.(meta_v1.Object), rhsObj.(meta_v1.Object))
}

// InformerProvider implements clusterpolicycontroller.Module
func (s *PodSecurityPolicies) InformerProvider() controller_informers.InformerProvider {
	return s.informers.Extensions().V1beta1().PodSecurityPolicies()
}

// Instance implements clusterpolicycontroller.Module
func (s *PodSecurityPolicies) Instance() meta_v1.Object {
	return &v1beta.PodSecurityPolicy{}
}

// Extract implements clusterpolicycontroller.Module
func (s *PodSecurityPolicies) Extract(clusterPolicy *policyhierarchy_v1.ClusterPolicy) []meta_v1.Object {
	var policies []runtime.Object
	for _, p := range clusterPolicy.Spec.PodSecurityPoliciesV1Beta1 {
		policies = append(policies, p.DeepCopy())
	}
	return object.RuntimeToMeta(policies)
}

// ActionSpec implements clusterpolicycontroller.Module
func (s *PodSecurityPolicies) ActionSpec() *action.ReflectiveActionSpec {
	return action.NewSpec(
		&v1beta.PodSecurityPolicy{},
		v1beta.SchemeGroupVersion,
		s.equalSpec,
		s.client.ExtensionsV1beta1(),
		s.informers.Extensions().V1beta1().PodSecurityPolicies().Lister(),
	)
}
