package controllers

import rbacv1 "k8s.io/api/rbac/v1"

// rolereference returns an intialized Role with apigroup, kind and name.
func rolereference(name, kind string) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     kind,
		Name:     name,
	}
}

// subject returns and initialized Subject with kind, name and namespace.
func subject(name, namespace, kind string) rbacv1.Subject {
	return rbacv1.Subject{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
	}
}
