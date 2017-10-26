package authorizer

import (
	"strings"
	"testing"

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	authz "k8s.io/api/authorization/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type testAuthorizeTestCase struct {
	// The content of the policy node storage in the test.
	storage []runtime.Object

	// The reviews that should pass.
	shouldPass []authz.SubjectAccessReviewSpec
	// The reviews that should fail.
	shouldFail []authz.SubjectAccessReviewSpec
}

// requests is a shorthand type.
type requests []authz.SubjectAccessReviewSpec

// TestAuthorizeHierarchical tests the hierarchical policy evaluation.  We
// don't necessarily test various authorization varieties since the
// implementation simply delegates to the existing RBAC authorizer once the
// hierarchical policies are collected.
func TestAuthorizeHierarchical(t *testing.T) {
	tests := []testAuthorizeTestCase{
		// Test basic authorization.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				policyNode(
					"kitties", "",
					[]rbac.Role{
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:jane"),
					}),
			},
			shouldPass: requests{
				request("jane", "meowie", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				request("jane", "meowie", "kitties", "pods", "list"),
			},
		},
		// Test basic hierarchical authorization: the roles and role bindings
		// are in the parent namespace.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				policyNode(
					"animals", "",
					[]rbac.Role{
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:jane"),
					}),
				policyNode("kitties", "animals", []rbac.Role{}, []rbac.RoleBinding{}),
			},
			shouldPass: requests{
				request("jane", "meowie", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				request("jane", "meowie", "kitties", "pods", "list"),
			},
		},
		// Role are redefined in the child namespace.  Those are ignored in our
		// current model.
		//
		// -- animals
		//    | pod-reader: ["get"], ["pods"]
		//    | role-binding: "jane" -> pod-reader
		//    |
		//    ` kitties
		//        pod-reader: ["list"], ["pods"]
		testAuthorizeTestCase{
			storage: []runtime.Object{
				policyNode("animals", "",
					[]rbac.Role{
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:jane"),
					}),
				policyNode("kitties", "animals",
					[]rbac.Role{
						// This role definition is overridden by the role definition in the parent.
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{}),
			},
			shouldPass: requests{
				request("jane", "meowie", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				request("jane", "meowie", "kitties", "pods", "list"),
			},
		},
		// Added a role binding and a role in the subordinate namespace.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				policyNode("animals", "",
					[]rbac.Role{
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:jane"),
					}),
				policyNode("kitties", "animals",
					[]rbac.Role{
						// This role definition is overridden by the role definition in the parent.
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:joe"),
					}),
			},
			shouldPass: requests{
				// Allowed by the policy and binding from namespace "animals".
				request("jane", "meowie", "kitties", "pods", "get"),
				// Allowed by the policy and bin.
				request("joe", "meowie", "kitties", "pods", "get"),
				// The policy set up above will allow any pod, so "anypod" will
				// work.
				request("jane", "anypod", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				// These fail because no users are allowed to do "list".
				request("jane", "meowie", "kitties", "pods", "list"),
				request("joe", "meowie", "kitties", "pods", "list"),
				// All of these have at least one mismatch with respect to the
				// policies defined.
				request("unknown", "meowie", "kitties", "pods", "get"),
				request("jane", "meowie", "unknown", "pods", "get"),
				request("jane", "meowie", "kitties", "unknown", "get"),
				request("jane", "meowie", "kitties", "pods", "unknown"),
			},
		},
		// Checks the effect of role bindings across the policy nodes.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				policyNode("life", "",
					[]rbac.Role{
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:jane"),
						roleBinding("pod-lister", "User:jane"),
						roleBinding("pod-watcher", "User:jane"),
					}),
				policyNode("animals", "life",
					[]rbac.Role{
						role("pod-lister",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:joe"),
						roleBinding("pod-lister", "User:joe"),
						roleBinding("pod-watcher", "User:joe"),
					}),
				policyNode("kitties", "animals",
					[]rbac.Role{
						role("pod-watcher",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"watch"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("pod-reader", "User:jill"),
						roleBinding("pod-lister", "User:jill"),
						roleBinding("pod-watcher", "User:jill"),
					}),
			},
			shouldPass: requests{
				request("jane", "meowie", "life", "pods", "get"),
				request("joe", "meowie", "animals", "pods", "get"),
				request("joe", "meowie", "animals", "pods", "get"),
				// Jill is a getter, lister, and watcaher in namespace "kitties".
				request("jill", "meowie", "kitties", "pods", "get"),
				request("jill", "meowie", "kitties", "pods", "get"),
				request("jill", "meowie", "kitties", "pods", "get"),
				request("jane", "meowie", "life", "pods", "get"),
			},
			shouldFail: requests{
				// Jane is not a a lister, nor a watcher in "life".
				request("jane", "meowie", "life", "pods", "list"),
				request("jane", "meowie", "life", "pods", "watch"),
				// Joe is not a watcher in "animals".
				request("joe", "meowie", "animals", "pods", "watch"),
				// Neither Joe nor Jill are getters in the namespace "life".
				// They are in "animals", but that role binding is not valid
				// in "life".
				request("joe", "meowie", "life", "pods", "get"),
				request("jill", "meowie", "life", "pods", "get"),
			},
		},
		testAuthorizeTestCase{
			storage: []runtime.Object{
				// The "animals" policy node is missing!
				policyNode("kitties", "animals",
					[]rbac.Role{
						role("pod-reader",
							[]rbac.PolicyRule{
								policyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{}),
			},
			shouldFail: requests{
				// User Jane is not allowed because the role binding is in
				// the policy node that currently does not seem to exist.
				request("jane", "meowie", "kitties", "pods", "list"),
				request("jane", "meowie", "kitties", "pods", "get"),
			},
		},
	}
	for i, tt := range tests {
		a := New(fakeinformers.NewPolicyNodeInformer(tt.storage...).Informer())

		for j, e := range tt.shouldPass {
			actual := a.Authorize(&e)
			if !actual.Allowed {
				t.Errorf("[%v->%v] should pass but didn't: %+v, was: %+v",
					i, j, e, actual)
			}
		}

		for j, e := range tt.shouldFail {
			actual := a.Authorize(&e)
			if actual.Allowed {
				t.Errorf("[%v->%v] should fail but didn't: %+v, was: %+v",
					i, j, e, actual)
			}
		}
	}
}

// policyNode prepares a policy node with given parent and name, and the
// preconfigured role and role bindings.  Set parent to "" for a policy node
// without a defined parent.
func policyNode(
	name, parent string,
	roles []rbac.Role,
	roleBindings []rbac.RoleBinding) *v1.PolicyNode {
	ret := v1.PolicyNode{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{
			Parent: parent,
			Policies: v1.PolicyLists{
				Roles:        roles,
				RoleBindings: roleBindings,
			},
		},
	}
	return &ret
}

// Example: roleBinding("pod-reader", "User:jane")
func roleBinding(roleName string, subjects ...string) rbac.RoleBinding {
	subjectList := []rbac.Subject{}
	for _, subject := range subjects {
		// "User:joe" -> ["User", "joe"]
		s := strings.Split(subject, ":")
		r := rbac.Subject{
			Kind:     s[0],
			Name:     s[1],
			APIGroup: "rbac.authorization.k8s.io",
		}
		subjectList = append(subjectList, r)
	}
	ret := rbac.RoleBinding{
		RoleRef:  roleRef(roleName),
		Subjects: subjectList,
	}
	return ret
}

// Example: roleRef("pod-reader")
func roleRef(name string) rbac.RoleRef {
	return rbac.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		// We currently don't do ClusterRole at all.
		Kind: "Role",
		Name: name,
	}
}

// Example:
// role("pod-reader",
//   []rbac.PolicyRule{
//     policyRule([]string{""}, []string{"get"}, []string{"pods"}),
// }),
func role(name string, policyRules []rbac.PolicyRule) rbac.Role {
	return rbac.Role{
		ObjectMeta: meta.ObjectMeta{
			Name: name,
		},
		Rules: policyRules,
	}
}

// Example:
// policyRule([]string{"someapigroup"}, []string{"get"}, []string{"pods"}),
func policyRule(apiGroups, verbs, resources []string) rbac.PolicyRule {
	return rbac.PolicyRule{
		APIGroups: apiGroups,
		Verbs:     verbs,
		Resources: resources,
	}
}

// Example:
// request("jane", "meowie", "kitties", "pods", "get").
func request(user, resourceName, namespace, resourceType, verb string,
) authz.SubjectAccessReviewSpec {
	return authz.SubjectAccessReviewSpec{
		ResourceAttributes: &authz.ResourceAttributes{
			Name:      resourceName,
			Namespace: namespace,
			Resource:  resourceType,
			Verb:      verb,
		},
		User: user,
	}
}
