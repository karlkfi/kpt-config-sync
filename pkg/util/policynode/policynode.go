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

package policynode

import (
	"github.com/google/stolos/pkg/util/namespaceutil"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
