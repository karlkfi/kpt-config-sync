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

// The algorithms to evaluate policies stored in PolicyTree.

package flattening

import (
	"github.com/google/nomos/pkg/util/set/stringset"
)

// flatten computes a non-hierarchical representation of RBAC policies, taking
// into account the parent and child policies.
func flatten(name nodeName, parent, child Policy) Policy {
	result := NewPolicy()

	seenRoles := stringset.New()
	for _, policy := range []Policy{parent, child} {
		// Flatten RoleBindings
		for _, policyItem := range policy.RoleBindings() {
			newItem := policyItem.DeepCopy()
			newItem.Namespace = string(name)
			result.AddRoleBinding(*newItem)
		}
		// Flatten Roles.
		for _, policyItem := range policy.Roles() {
			if seenRoles.Contains(policyItem.Name) {
				// This named role has already been seen.  Skip it.
				// Since we process parent roles first, this will make
				// the role definition from the parent policy node win.
				continue
			}
			newItem := policyItem.DeepCopy()
			newItem.Namespace = string(name)
			result.AddRole(*newItem)
			seenRoles.Add(policyItem.Name)
		}
	}
	return *result
}
