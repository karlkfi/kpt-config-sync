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

package flattening

import (
	rbac "k8s.io/api/rbac/v1"
)

// Policy is the stored policy content.
type Policy struct {
	// The list of role bindings contained in this policy.
	roleBindings []rbac.RoleBinding
	// The list of roles contained in this policy.
	roles []rbac.Role
}

// NewPolicy creates a new empty policy.  Use one of the setter methods to
// set, or add new content to the policy.
func NewPolicy() *Policy {
	return &Policy{[]rbac.RoleBinding{}, []rbac.Role{}}
}

// SetRoleBindings sets the role bindings on this policy.
func (p *Policy) SetRoleBindings(r ...rbac.RoleBinding) *Policy {
	p.roleBindings = r
	return p
}

// SetRoles sets the roles on this policy.  Returns resulting policy.
func (p *Policy) SetRoles(r ...rbac.Role) *Policy {
	p.roles = r
	return p
}

// AddRoleBinding adds the given role binding to the bindings already contained
// in this policy.  No checking is done.  Returns resulting policy.
func (p *Policy) AddRoleBinding(r rbac.RoleBinding) *Policy {
	p.roleBindings = append(p.roleBindings, r)
	return p
}

// AddRole adds the given role to the roles already contained in this policy.
// Returns resulting policy.
func (p *Policy) AddRole(r rbac.Role) *Policy {
	p.roles = append(p.roles, r)
	return p
}

// RoleBindings returns the stored role bindings.
func (p *Policy) RoleBindings() []rbac.RoleBinding {
	return p.roleBindings
}

// Roles returns the stored roles.
func (p *Policy) Roles() []rbac.Role {
	return p.roles
}
