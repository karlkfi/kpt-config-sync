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

package v1

import "github.com/pkg/errors"

// Validate returns an error if the ClusterPolicy contains sub-resources with
// duplicate names.
func (c *ClusterPolicy) Validate() error {
	clusterRoleNames := make(map[string]bool)
	for _, r := range c.Spec.ClusterRolesV1 {
		if n := r.Name; clusterRoleNames[n] {
			return errors.Errorf("duplicate clusterrole name %q in clusterpolicy", n)
		}
	}
	clusterRoleBindingNames := make(map[string]bool)
	for _, rb := range c.Spec.ClusterRoleBindingsV1 {
		if n := rb.Name; clusterRoleBindingNames[n] {
			return errors.Errorf("duplicate clusterrolebinding name %q in clusterpolicy", n)
		}
	}
	podSecurityPolicyNames := make(map[string]bool)
	for _, psp := range c.Spec.PodSecurityPoliciesV1Beta1 {
		if n := psp.Name; podSecurityPolicyNames[n] {
			return errors.Errorf("duplicate podsecuritypolicy name %q in clusterpolicy", n)
		}
	}

	return nil
}
