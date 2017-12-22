package authorizer

import (
	"testing"

	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	"github.com/google/stolos/pkg/testing/rbactesting"
	authz "k8s.io/api/authorization/v1beta1"
	rbac "k8s.io/api/rbac/v1"
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
			input: rbactesting.Request("jane", "meowie", "kitties", "pods", "get"),
			expectedIsResourceRequest: true,
			expectedNamespace:         "kitties",
		},
		{
			// GetNamespace() call should succeed even on a non-resource request.
			input: rbactesting.NonResourceRequest("jane", "/some/path", "get"),
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
				rbactesting.PolicyNode(
					"kitties", "",
					[]rbac.Role{
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:jane"),
					}),
			},
			shouldPass: requests{
				rbactesting.Request("jane", "meowie", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				rbactesting.Request("jane", "meowie", "kitties", "pods", "list"),
			},
		},
		// Test basic hierarchical authorization: the roles and role bindings
		// are in the parent namespace.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				rbactesting.PolicyNode(
					"animals", "",
					[]rbac.Role{
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:jane"),
					}),
				rbactesting.PolicyNode("kitties", "animals", []rbac.Role{}, []rbac.RoleBinding{}),
			},
			shouldPass: requests{
				rbactesting.Request("jane", "meowie", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				rbactesting.Request("jane", "meowie", "kitties", "pods", "list"),
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
				rbactesting.PolicyNode("animals", "",
					[]rbac.Role{
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:jane"),
					}),
				rbactesting.PolicyNode("kitties", "animals",
					[]rbac.Role{
						// This role definition is overridden by the role definition in the parent.
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{}),
			},
			shouldPass: requests{
				rbactesting.Request("jane", "meowie", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				rbactesting.Request("jane", "meowie", "kitties", "pods", "list"),
			},
		},
		// Added a role binding and a role in the subordinate namespace.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				rbactesting.PolicyNode("animals", "",
					[]rbac.Role{
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:jane"),
					}),
				rbactesting.PolicyNode("kitties", "animals",
					[]rbac.Role{
						// This role definition is overridden by the role definition in the parent.
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:joe"),
					}),
			},
			shouldPass: requests{
				// Allowed by the policy and binding from namespace "animals".
				rbactesting.Request("jane", "meowie", "kitties", "pods", "get"),
				// Allowed by the policy and bin.
				rbactesting.Request("joe", "meowie", "kitties", "pods", "get"),
				// The policy set up above will allow any pod, so "anypod" will
				// work.
				rbactesting.Request("jane", "anypod", "kitties", "pods", "get"),
			},
			shouldFail: requests{
				// These fail because no users are allowed to do "list".
				rbactesting.Request("jane", "meowie", "kitties", "pods", "list"),
				rbactesting.Request("joe", "meowie", "kitties", "pods", "list"),
				// All of these have at least one mismatch with respect to the
				// policies defined.
				rbactesting.Request("unknown", "meowie", "kitties", "pods", "get"),
				rbactesting.Request("jane", "meowie", "unknown", "pods", "get"),
				rbactesting.Request("jane", "meowie", "kitties", "unknown", "get"),
				rbactesting.Request("jane", "meowie", "kitties", "pods", "unknown"),
			},
		},
		// Checks the effect of role bindings across the policy nodes.
		testAuthorizeTestCase{
			storage: []runtime.Object{
				rbactesting.PolicyNode("life", "",
					[]rbac.Role{
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"get"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:jane"),
						rbactesting.RoleBinding("pod-lister", "User:jane"),
						rbactesting.RoleBinding("pod-watcher", "User:jane"),
					}),
				rbactesting.PolicyNode("animals", "life",
					[]rbac.Role{
						rbactesting.Role("pod-lister",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:joe"),
						rbactesting.RoleBinding("pod-lister", "User:joe"),
						rbactesting.RoleBinding("pod-watcher", "User:joe"),
					}),
				rbactesting.PolicyNode("kitties", "animals",
					[]rbac.Role{
						rbactesting.Role("pod-watcher",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"watch"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("pod-reader", "User:jill"),
						rbactesting.RoleBinding("pod-lister", "User:jill"),
						rbactesting.RoleBinding("pod-watcher", "User:jill"),
					}),
			},
			shouldPass: requests{
				rbactesting.Request("jane", "meowie", "life", "pods", "get"),
				rbactesting.Request("joe", "meowie", "animals", "pods", "get"),
				rbactesting.Request("joe", "meowie", "animals", "pods", "get"),
				// Jill is a getter, lister, and watcher in namespace "kitties".
				rbactesting.Request("jill", "meowie", "kitties", "pods", "get"),
				rbactesting.Request("jill", "meowie", "kitties", "pods", "get"),
				rbactesting.Request("jill", "meowie", "kitties", "pods", "get"),
				rbactesting.Request("jane", "meowie", "life", "pods", "get"),
			},
			shouldFail: requests{
				// Jane is not a a lister, nor a watcher in "life".
				rbactesting.Request("jane", "meowie", "life", "pods", "list"),
				rbactesting.Request("jane", "meowie", "life", "pods", "watch"),
				// Joe is not a watcher in "animals".
				rbactesting.Request("joe", "meowie", "animals", "pods", "watch"),
				// Neither Joe nor Jill are getters in the namespace "life".
				// They are in "animals", but that role binding is not valid
				// in "life".
				rbactesting.Request("joe", "meowie", "life", "pods", "get"),
				rbactesting.Request("jill", "meowie", "life", "pods", "get"),
			},
		},
		testAuthorizeTestCase{
			storage: []runtime.Object{
				// The "animals" policy node is missing!
				rbactesting.PolicyNode("kitties", "animals",
					[]rbac.Role{
						rbactesting.Role("pod-reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule([]string{""}, []string{"list"}, []string{"pods"}),
							}),
					},
					[]rbac.RoleBinding{}),
			},
			shouldFail: requests{
				// User Jane is not allowed because the role binding is in
				// the policy node that currently does not seem to exist.
				rbactesting.Request("jane", "meowie", "kitties", "pods", "list"),
				rbactesting.Request("jane", "meowie", "kitties", "pods", "get"),
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
				rbactesting.PolicyNode("acme", "",
					[]rbac.Role{
						rbactesting.NamespaceRole("admin", "acme",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule(
									[]string{""},
									[]string{"*"},
									[]string{"pods"},
								),
								rbactesting.PolicyRule(
									[]string{""},
									[]string{"get"},
									[]string{"namespaces", "roles", "rolebindings"},
								),
							}),
					},
					[]rbac.RoleBinding{}),
				rbactesting.PolicyNode("eng", "acme",
					[]rbac.Role{},
					[]rbac.RoleBinding{
						rbactesting.NamespaceRoleBinding("alice-rolebinding", "eng", "admin",
							"User:alice@google.com"),
					}),
				rbactesting.PolicyNode("corp", "acme", []rbac.Role{}, []rbac.RoleBinding{}),
				rbactesting.PolicyNode("rnd", "acme", []rbac.Role{}, []rbac.RoleBinding{}),
				rbactesting.PolicyNode("new-prj", "rnd", []rbac.Role{}, []rbac.RoleBinding{}),
				rbactesting.PolicyNode("backend", "eng", []rbac.Role{}, []rbac.RoleBinding{
					rbactesting.NamespaceRoleBinding("bob-rolebinding", "backend", "admin",
						"User:bob@google.com"),
				}),
				rbactesting.PolicyNode("frontend", "eng", []rbac.Role{}, []rbac.RoleBinding{}),
			},
			shouldPass: requests{
				// alice@google.com can do all the operations that admin is
				// allowed to do.
				rbactesting.Request("alice@google.com", "", "eng", "pods", "get"),
				rbactesting.Request("alice@google.com", "", "eng", "namespaces", "get"),
				rbactesting.Request("alice@google.com", "", "eng", "roles", "get"),
				rbactesting.Request("alice@google.com", "", "eng", "rolebindings", "get"),
				rbactesting.Request("alice@google.com", "", "backend", "rolebindings", "get"),
				rbactesting.Request("alice@google.com", "", "frontend", "rolebindings", "get"),

				// bob@google.com as admin in "backend" can do all the admin ops
				// like alice@, but in its namespace only.
				rbactesting.Request("bob@google.com", "", "backend", "rolebindings", "get"),
				rbactesting.Request("bob@google.com", "", "backend", "namespaces", "get"),
				rbactesting.Request("bob@google.com", "", "backend", "roles", "get"),
				rbactesting.Request("bob@google.com", "", "backend", "rolebindings", "get"),
				// ...and operate on pods every which way.
				rbactesting.Request("bob@google.com", "", "backend", "pods", "get"),
				rbactesting.Request("bob@google.com", "", "backend", "pods", "create"),
				rbactesting.Request("bob@google.com", "", "backend", "pods", "list"),
			},
			shouldFail: requests{
				// alice@google.com has no permissions in 'corp' and 'rnd.
				rbactesting.Request("alice@google.com", "", "corp", "rolebindings", "get"),
				rbactesting.Request("alice@google.com", "", "rnd", "rolebindings", "get"),
				// bob@google.com is not admin in frontend, but in backend.
				rbactesting.Request("bob@google.com", "", "frontend", "rolebindings", "get"),
				// bob@google.com is not admin in eng.
				rbactesting.Request("bob@google.com", "", "eng", "rolebindings", "get"),
				// bob@google.com is admin, but admins can not 'create'.
				rbactesting.Request("bob@google.com", "", "frontend", "rolebindings", "create"),
			},
		},
		testAuthorizeTestCase{
			// A permission for an operation on resourcequotas in the core API
			// group implies that same permission on
			// k8us.k8s.io/stolosresourcequotas
			storage: []runtime.Object{
				rbactesting.PolicyNode("kitties", "",
					[]rbac.Role{
						rbactesting.Role("reader",
							[]rbac.PolicyRule{
								rbactesting.PolicyRule(
									[]string{""},
									[]string{"list"},
									[]string{"resourcequotas"}),
							}),
					},
					[]rbac.RoleBinding{
						rbactesting.RoleBinding("reader", "User:jane"),
					}),
			},
			shouldPass: requests{
				// This is what the policy in "reader" says verbatim.
				rbactesting.Request("jane", "", "kitties", "resourcequotas", "list"),
				// This permission is derived from the permission to list
				// "regular" resource quotas.
				rbactesting.RequestWithGroup(
					"jane", "", "kitties",
					rbactesting.ResourceGroup(policyhierarchy.GroupName),
					policyhierarchy.StolosResourceQuotaResource, "list"),
			},
			shouldFail: requests{
				// Forbidden because the reader role doesn't have create
				// permissions.
				rbactesting.Request("jane", "", "kitties", "resourcequotas", "create"),
				// And there is nothing to grant a similar permission for Stolos
				// resource quotas either.
				rbactesting.RequestWithGroup(
					"jane", "", "kitties",
					rbactesting.ResourceGroup(policyhierarchy.GroupName),
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
