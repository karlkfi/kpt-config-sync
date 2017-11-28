package authorizer

import (
	"strings"
	"testing"

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	authz "k8s.io/api/authorization/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestAttributes tests the implicit contracts on Attributes method that are
// not formally documented, but are a consequence of the way the K8S RBAC
// authorizer works.
func TestAttributes(t *testing.T) {
	tests := []struct {
		input                     authz.SubjectAccessReviewSpec
		expectedIsResourceRequest bool
		expectedNamespace         string
	}{
		{
			// Base case.
			input: request("jane", "meowie", "kitties", "pods", "get"),
			expectedIsResourceRequest: true,
			expectedNamespace:         "kitties",
		},
		{
			// GetNamespace() call should succeed even on a non-resource request.
			input: nonResourceRequest("jane", "/some/path", "get"),
			expectedIsResourceRequest: false,
			expectedNamespace:         "",
		},
	}
	for i, tt := range tests {
		attr := NewAttributes(&tt.input)
		isResourceRequest := attr.IsResourceRequest()
		if isResourceRequest != tt.expectedIsResourceRequest {
			t.Errorf("[%v] IsResourceRequest mismatch: expected: %v, actual: %v for: %v",
				i, tt.expectedIsResourceRequest, isResourceRequest, tt.input)
		}
		if attr.GetNamespace() != tt.expectedNamespace {
			t.Errorf("[%v] GetNamespace mismatch: expected: %v, actual: %v for: %v",
				i, tt.expectedNamespace, attr.GetNamespace(), tt.input)
		}
	}
}

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
				// Jill is a getter, lister, and watcher in namespace "kitties".
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
		testAuthorizeTestCase{
			// The canonical "acme corp" test case.
			//
			// Note that APIGroup in the policy rule must be filled in with
			// something, otherwise that policy rule is effectively defunct.
			//
			// acme (Admin: pod.*)
			// + corp
			// + eng (alice@google.com -> Admin)
			// |  + backend (bob@google.com -> Admin)
			// |  ` frontend
			// ` rnd
			//   ` new-prj
			storage: []runtime.Object{
				policyNode("acme", "",
					[]rbac.Role{
						namespaceRole("admin", "acme",
							[]rbac.PolicyRule{
								policyRule(
									[]string{""},
									[]string{"*"},
									[]string{"pods"},
								),
								policyRule(
									[]string{""},
									[]string{"get"},
									[]string{"namespaces", "roles", "rolebindings"},
								),
							}),
					},
					[]rbac.RoleBinding{}),
				policyNode("eng", "acme",
					[]rbac.Role{},
					[]rbac.RoleBinding{
						namespaceRoleBinding("alice-rolebinding", "eng", "admin",
							"User:alice@google.com"),
					}),
				policyNode("corp", "acme", []rbac.Role{}, []rbac.RoleBinding{}),
				policyNode("rnd", "acme", []rbac.Role{}, []rbac.RoleBinding{}),
				policyNode("new-prj", "rnd", []rbac.Role{}, []rbac.RoleBinding{}),
				policyNode("backend", "eng", []rbac.Role{}, []rbac.RoleBinding{
					namespaceRoleBinding("bob-rolebinding", "backend", "admin",
						"User:bob@google.com"),
				}),
				policyNode("frontend", "eng", []rbac.Role{}, []rbac.RoleBinding{}),
			},
			shouldPass: requests{
				// alice@google.com can do all the operations that admin is
				// allowed to do.
				request("alice@google.com", "", "eng", "pods", "get"),
				request("alice@google.com", "", "eng", "namespaces", "get"),
				request("alice@google.com", "", "eng", "roles", "get"),
				request("alice@google.com", "", "eng", "rolebindings", "get"),
				request("alice@google.com", "", "backend", "rolebindings", "get"),
				request("alice@google.com", "", "frontend", "rolebindings", "get"),

				// bob@google.com as admin in "backend" can do all the admin ops
				// like alice@, but in its namespace only.
				request("bob@google.com", "", "backend", "rolebindings", "get"),
				request("bob@google.com", "", "backend", "namespaces", "get"),
				request("bob@google.com", "", "backend", "roles", "get"),
				request("bob@google.com", "", "backend", "rolebindings", "get"),
				// ...and operate on pods every which way.
				request("bob@google.com", "", "backend", "pods", "get"),
				request("bob@google.com", "", "backend", "pods", "create"),
				request("bob@google.com", "", "backend", "pods", "list"),
			},
			shouldFail: requests{
				// alice@google.com has no permissions in 'corp' and 'rnd.
				request("alice@google.com", "", "corp", "rolebindings", "get"),
				request("alice@google.com", "", "rnd", "rolebindings", "get"),
				// bob@google.com is not admin in frontend, but in backend.
				request("bob@google.com", "", "frontend", "rolebindings", "get"),
				// bob@google.com is not admin in eng.
				request("bob@google.com", "", "eng", "rolebindings", "get"),
				// bob@google.com is admin, but admins can not 'create'.
				request("bob@google.com", "", "frontend", "rolebindings", "create"),
			},
		},
		testAuthorizeTestCase{
			// A permission for an operation on resourcequotas in the core API
			// group implies that same permission on
			// k8us.k8s.io/stolosresourcequotas
			storage: []runtime.Object{
				policyNode("kitties", "",
					[]rbac.Role{
						role("reader",
							[]rbac.PolicyRule{
								policyRule(
									[]string{""},
									[]string{"list"},
									[]string{"resourcequotas"}),
							}),
					},
					[]rbac.RoleBinding{
						roleBinding("reader", "User:jane"),
					}),
			},
			shouldPass: requests{
				// This is what the policy in "reader" says verbatim.
				request("jane", "", "kitties", "resourcequotas", "list"),
				// This permission is derived from the permission to list
				// "regular" resource quotas.
				requestWithGroup(
					"jane", "", "kitties",
					resourceGroup(policyhierarchy.GroupName),
					policyhierarchy.StolosResourceQuotaResource, "list"),
			},
			shouldFail: requests{
				// Forbidden because the reader role doesn't have create
				// permissions.
				request("jane", "", "kitties", "resourcequotas", "create"),
				// And there is nothing to grant a similar permission for Stolos
				// resource quotas either.
				requestWithGroup(
					"jane", "", "kitties",
					resourceGroup(policyhierarchy.GroupName),
					policyhierarchy.StolosResourceQuotaResource, "create"),
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

func TestMatchesCoreResourceQuota(t *testing.T) {
	tests := []struct {
		name      string
		apiGroups []string
		resources []string
		matches   bool
	}{
		{
			name:      "Matches explicit core api group",
			apiGroups: []string{""},
			resources: []string{"resourcequotas"},
			matches:   true,
		},
		{
			name:      "Matches a wildcard",
			apiGroups: []string{"*"},
			resources: []string{"resourcequotas"},
			matches:   true,
		},
		{
			name:      "Does not match an empty API group",
			apiGroups: []string{},
			resources: []string{"resourcequotas"},
			matches:   false,
		},
		{
			name:      "Does not match empty resources",
			apiGroups: []string{""},
			resources: []string{},
			matches:   false,
		},
		{
			name:      "Does not match empty resources",
			apiGroups: []string{""},
			resources: []string{},
			matches:   false,
		},
		{
			name:      "Does not match mismatching resources",
			apiGroups: []string{"group1", "group2"},
			resources: []string{"resource1", "resource2"},
			matches:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matchesCoreResourceQuota(tt.apiGroups, tt.resources)
			if matches != tt.matches {
				t.Errorf("Expected: %v, got: %v, for: %#v",
					tt.matches, matches, tt)
			}
		})
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
			Policies: v1.Policies{
				Roles:        roles,
				RoleBindings: roleBindings,
			},
		},
	}
	return &ret
}

// Example: roleBinding("pod-reader", "User:jane")
func roleBinding(roleName string, subjects ...string) rbac.RoleBinding {
	return namespaceRoleBinding("", "", roleName, subjects...)
}

func namespaceRoleBinding(name, namespace, roleName string, subjects ...string) rbac.RoleBinding {
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
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
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
	return namespaceRole(name, "", policyRules)
}

func namespaceRole(name, namespace string, policyRules []rbac.PolicyRule) rbac.Role {
	return rbac.Role{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: policyRules,
	}

}

// policyRule creates a new component of a Role.
//
// Note that apiGroups must be set to a non-empty slice in order for it to ever
// take effect.  This is a tacit requirement of the native K8S RBAC authz.
// Example:
// policyRule([]string{"someapigroup"}, []string{"get"}, []string{"pods"}),
func policyRule(apiGroups, verbs, resources []string) rbac.PolicyRule {
	if len(apiGroups) == 0 {
		panic("apiGroups must have at least one element to be effective.")
	}
	return rbac.PolicyRule{
		APIGroups: apiGroups,
		Verbs:     verbs,
		Resources: resources,
	}
}

// Used to disambiguate arguments.
type resourceGroup string

// Example:
// request("jane", "meowie", "kitties", "pods", "get").
func request(user, resourceName, namespace, resourceType, verb string,
) authz.SubjectAccessReviewSpec {
	return requestWithGroup(user, resourceName, namespace, resourceGroup(""), resourceType, verb)
}

func requestWithGroup(user, resourceName, namespace string,
	group resourceGroup, resourceType, verb string) authz.SubjectAccessReviewSpec {
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

// Example:
//   nonResourceRequest("jane", "/some/path", "get).
func nonResourceRequest(user, path, verb string) authz.SubjectAccessReviewSpec {
	return authz.SubjectAccessReviewSpec{
		NonResourceAttributes: &authz.NonResourceAttributes{
			Path: path,
			Verb: verb,
		},
		User: user,
	}
}
