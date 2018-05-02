package rbactesting

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	authz "k8s.io/api/authorization/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PolicyNode prepares a policy node with given parent and name, and the
// preconfigured role and role bindings.  Set parent to "" for a policy node
// without a defined parent.
func PolicyNode(
	name, parent string, policyspace bool,
	roles []rbac.Role,
	roleBindings []rbac.RoleBinding) *v1.PolicyNode {
	var policyNodeType v1.PolicyNodeType
	if policyspace {
		policyNodeType = v1.Policyspace
	} else {
		policyNodeType = v1.Namespace
	}
	ret := v1.PolicyNode{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{
			Type:   policyNodeType,
			Parent: parent,
			Policies: v1.Policies{
				RolesV1:        roles,
				RoleBindingsV1: roleBindings,
			},
		},
	}
	return &ret
}

// RoleBinding is a convenience function for creating a RoleBinding, example:
// RoleBinding("pod-reader", "User:jane")
func RoleBinding(roleName string, subjects ...string) rbac.RoleBinding {
	return NamespaceRoleBinding("", "", roleName, subjects...)
}

// NamespaceRoleBinding is a convenience function for creating a RoleBinding with namespace set.
func NamespaceRoleBinding(name, namespace, roleName string, subjects ...string) rbac.RoleBinding {
	subjectList := []rbac.Subject{}
	var kind string
	group := "rbac.authorization.k8s.io"
	for _, subject := range subjects {
		// "User:joe" -> ["User", "joe"]
		s := strings.Split(subject, ":")
		if len(s) != 2 {
			panic(fmt.Sprintf("Expected subject: User:name, was: %v", subject))
		}
		kind = s[0]
		r := rbac.Subject{
			Kind:     s[0],
			Name:     s[1],
			APIGroup: group,
		}
		subjectList = append(subjectList, r)
	}
	ret := rbac.RoleBinding{
		TypeMeta: meta.TypeMeta{
			Kind:       kind,
			APIVersion: group + "/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef:  RoleRef(roleName),
		Subjects: subjectList,
	}
	return ret
}

// RoleRef constructs a role ref to a name Example: RoleRef("pod-reader")
func RoleRef(name string) rbac.RoleRef {
	return rbac.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		// We currently don't do ClusterRole at all.
		Kind: "Role",
		Name: name,
	}
}

// Role creates a Role.
// Example:
// Role("pod-reader",
//   []rbac.PolicyRule{
//     PolicyRule([]string{""}, []string{"get"}, []string{"pods"}),
// }),
func Role(name string, policyRules []rbac.PolicyRule) rbac.Role {
	return NamespaceRole(name, "", policyRules)
}

// NamespaceRole creates a role with namespace set.
func NamespaceRole(name, namespace string, policyRules []rbac.PolicyRule) rbac.Role {
	return rbac.Role{
		TypeMeta: meta.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: policyRules,
	}

}

// PolicyRule creates a new component of a Role.
//
// Note that apiGroups must be set to a non-empty slice in order for it to ever
// take effect.  This is a tacit requirement of the native K8S RBAC authz.
// Example:
// PolicyRule([]string{"someapigroup"}, []string{"get"}, []string{"pods"}),
func PolicyRule(apiGroups, verbs, resources []string) rbac.PolicyRule {
	if len(apiGroups) == 0 {
		panic("apiGroups must have at least one element to be effective.")
	}
	return rbac.PolicyRule{
		APIGroups: apiGroups,
		Verbs:     verbs,
		Resources: resources,
	}
}

// ResourceGroup is a tinytype used to disambiguate arguments.
type ResourceGroup string

// Request creates a new SubjectAccessReviewSpec
// Example:
// Request("jane", "meowie", "kitties", "pods", "get").
func Request(user, resourceName, namespace, resourceType, verb string,
) authz.SubjectAccessReviewSpec {
	return RequestWithGroup(user, resourceName, namespace, ResourceGroup(""), resourceType, verb)
}

// RequestWithGroup creates a new SubjectAccessReviewSpec with group set.
func RequestWithGroup(user, resourceName, namespace string,
	group ResourceGroup, resourceType, verb string) authz.SubjectAccessReviewSpec {
	return authz.SubjectAccessReviewSpec{
		ResourceAttributes: &authz.ResourceAttributes{
			Name:      resourceName,
			Namespace: namespace,
			Resource:  resourceType,
			Verb:      verb,
			Group:     string(group),
		},
		User: user,
	}

}

// NonResourceRequest creates a SubjectAccessReviewSpec for a non-resource type.
// Example:
//   NonResourceRequest("jane", "/some/path", "get).
func NonResourceRequest(user, path, verb string) authz.SubjectAccessReviewSpec {
	return authz.SubjectAccessReviewSpec{
		NonResourceAttributes: &authz.NonResourceAttributes{
			Path: path,
			Verb: verb,
		},
		User: user,
	}
}
