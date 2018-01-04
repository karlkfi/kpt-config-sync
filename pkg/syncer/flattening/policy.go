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

// RoleBindings returns the stored roles.
func (p *Policy) Roles() []rbac.Role {
	return p.roles
}
