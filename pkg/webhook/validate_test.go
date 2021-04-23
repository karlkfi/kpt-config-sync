package webhook

import (
	"context"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/testing/fake"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestValidator_Handle(t *testing.T) {
	testCases := []struct {
		name   string
		oldObj client.Object
		newObj client.Object
		user   authenticationv1.UserInfo
		deny   metav1.StatusReason
	}{
		{
			// The Config Sync Importer is allowed to do anything it likes, so we just
			// have one test for it.
			name: "Importer creates a object whose configmanagement.gke.io/managed annotation is set to enabled but whose configsync.gke.io/resource-id annotation is unset",
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: configSyncImporter(),
		},
		{
			name: "Root reconciler deletes an object it manages",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceManagerKey, ":root"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{},"f:configsync.gke.io/manager":{}}},"f:rules":{}}`),
			),
			user: configSyncRootReconciler(),
		},
		{
			name: "Root reconciler deletes an object managed by a namespace reconciler",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceManagerKey, "bookstore"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{},"f:configsync.gke.io/manager":{}}},"f:rules":{}}`),
			),
			user: configSyncRootReconciler(),
		},
		{
			name: "Namespace reconciler deletes an object it manages",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceManagerKey, "bookstore"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{},"f:configsync.gke.io/manager":{}}},"f:rules":{}}`),
			),
			user: configSyncNamespaceReconciler("bookstore"),
		},
		{
			name: "Namespace reconciler deletes an object it does not manage",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceManagerKey, "videostore"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{},"f:configsync.gke.io/manager":{}}},"f:rules":{}}`),
			),
			user: configSyncNamespaceReconciler("bookstore"),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Namespace reconciler deletes an object it does not manage",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceManagerKey, "videostore"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{},"f:configsync.gke.io/manager":{}}},"f:rules":{}}`),
			),
			user: configSyncNamespaceReconciler("bookstore"),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Bob creates an unmanaged object, which does not have the configmanagement.gke.io/managed and configsync.gke.io/resource-id annotations.",
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob creates an unmanaged object, whose configmanagement.gke.io/managed annotation is set to enabled, but whose configsync.gke.io/resource-id annotation is unset.",
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob creates an unmanaged object, whose configmanagement.gke.io/managed annotation is unset, but whose configsync.gke.io/resource-id annotation is set.",
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob deletes an unmanaged object, which does not have the configmanagement.gke.io/managed and configsync.gke.io/resource-id annotations.",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob deletes an unmanaged object, whose configmanagement.gke.io/managed annotation is set to enabled, but whose configsync.gke.io/resource-id annotation is unset.",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob deletes an unmanaged object, whose configmanagement.gke.io/managed annotation is unset, but whose configsync.gke.io/resource-id annotation is set.",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob updates an unmanaged object, oldObj and newObj both do not have the configmanagement.gke.io/managed and configsync.gke.io/resource-id annotations.",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob updates an unmanaged object, oldObj has the configmanagement.gke.io/managed  annotation, newObj has the configsync.gke.io/resource-id annotation.",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			),
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				}),
			),
			user: bob(),
		},
		{
			name: "Bob creates a managed object",
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: bob(),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Bob deletes a managed object",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: bob(),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Bob updates a managed object: undeclared fields",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Label("acme.com/foo", "bar"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: bob(),
		},
		{
			name: "Bob updates a managed object: declared fields",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"*"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: bob(),
			deny: metav1.StatusReasonForbidden,
		},
		{
			name: "Bob updates a managed object: Config Sync metadata",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				// Removed managed-by label
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: bob(),
			deny: metav1.StatusReasonForbidden,
		},
		{
			name: "Bob updates a object (whose configmanagement.gke.io/managed annotation is unset, but whose configsync.gke.io/resource-id annotation is set): Config Sync metadata",
			oldObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			newObj: fake.RoleObject(
				core.Name("hello"),
				core.Namespace("world"),
				// Removed managed-by label
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				setRules([]rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
				core.Annotation(v1beta1.DeclaredFieldsKey, `{"f:metadata":{"f:labels":{"f:app.kubernetes.io/managed-by":{}},"f:annotations":{"f:configmanagement.gke.io/managed":{}}},"f:rules":{}}`),
			),
			user: bob(),
		},
		{
			name: "Bob creates a ResourceGroup generated by ConfigSync",
			newObj: fake.ResourceGroupObject(
				core.Name("repo-sync"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label(common.InventoryLabel, applier.InventoryID("bookstore"))),
			user: bob(),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Bob deletes a ResourceGroup generated by ConfigSync",
			oldObj: fake.ResourceGroupObject(
				core.Name("repo-sync"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label(common.InventoryLabel, applier.InventoryID("bookstore"))),
			user: bob(),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Bob updates a ResourceGroup generated by ConfigSync",
			oldObj: fake.ResourceGroupObject(
				core.Name("repo-sync"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label(common.InventoryLabel, applier.InventoryID("bookstore"))),
			newObj: fake.ResourceGroupObject(
				core.Name("repo-sync"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label(common.InventoryLabel, applier.InventoryID("bookstore")),
				core.Label("acme.com/foo", "bar")),
			user: bob(),
			deny: metav1.StatusReasonUnauthorized,
		},
		{
			name: "Bob creates an independent ResourceGroup",
			newObj: fake.ResourceGroupObject(
				core.Name("user-created"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			user: bob(),
		},
		{
			name: "Bob deletes an independent ResourceGroup",
			oldObj: fake.ResourceGroupObject(
				core.Name("user-created"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			user: bob(),
		},
		{
			name: "Bob updates an independent ResourceGroup",
			oldObj: fake.ResourceGroupObject(
				core.Name("user-created"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			newObj: fake.ResourceGroupObject(
				core.Name("user-created"),
				core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label("acme.com/foo", "bar")),
			user: bob(),
		},
		{
			name: "Bob manually modifies lifecycle annotation of an object, whose configmanagement.gke.io/managed annotation is set to enabled, but whose configsync.gke.io/resource-id annotation is unset",
			oldObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, v1beta1.IgnoreMutation)),
			newObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, "other")),
		},
		{
			name: "Bob manually modifies lifecycle annotation of an object, whose configsync.gke.io/resource-id annotation is incorrect",
			oldObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, v1beta1.IgnoreMutation)),
			newObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_world_hello"),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, "other")),
		},
		{
			name: "Bob manually modifies lifecycle annotation of a managed object",
			oldObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_default-name"),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, v1beta1.IgnoreMutation)),
			newObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_default-name"),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, "other")),
			deny: metav1.StatusReasonForbidden,
		},
		{
			name: "Bob manually adds lifecycle annotation",
			oldObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_default-name")),
			newObj: fake.RoleObject(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Annotation(v1beta1.ResourceIDKey, "rbac.authorization.k8s.io_role_default-name"),
				core.Annotation(v1beta1.LifecycleMutationAnnotation, v1beta1.IgnoreMutation)),
			deny: metav1.StatusReasonForbidden,
		},
	}

	v := validatorForTest(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := request(tc.oldObj, tc.newObj)
			req.UserInfo = tc.user

			resp := v.Handle(context.Background(), req)
			if resp.Allowed {
				if tc.deny != "" {
					t.Errorf("got Handle() response allowed, want denied %q", tc.deny)
				}
			} else if tc.deny == "" {
				t.Errorf("got Handle() response denied %q, want allowed", resp.Result.Reason)
			} else if tc.deny != resp.Result.Reason {
				t.Errorf("got Handle() response denied %q, want denied %q", resp.Result.Reason, tc.deny)
			}
		})
	}
}

func validatorForTest(t *testing.T) *Validator {
	vc, err := declared.ValueConverterForTest()
	if err != nil {
		t.Fatalf("Failed to create ValueConverter: %v", err)
	}
	od := &ObjectDiffer{converter: vc}
	return &Validator{differ: od}
}

func configSyncImporter() authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Groups:   []string{saGroup, saNamespaceGroup},
		Username: saImporter,
	}
}

func configSyncRootReconciler() authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Groups:   []string{saGroup, saNamespaceGroup},
		Username: saRootReconciler,
	}
}

func configSyncNamespaceReconciler(ns string) authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Groups:   []string{saGroup, saNamespaceGroup},
		Username: saNamespacePrefix + ns,
	}
}

func bob() authenticationv1.UserInfo {
	return authenticationv1.UserInfo{
		Groups:   []string{"devs@acme.com"},
		Username: "bob@acme.com",
	}
}

func request(oldObj, newObj client.Object) admission.Request {
	var gvk schema.GroupVersionKind
	var name, namespace string

	var operation admissionv1.Operation
	if oldObj == nil {
		operation = admissionv1.Create
		gvk = newObj.GetObjectKind().GroupVersionKind()
		name = newObj.GetName()
		namespace = newObj.GetNamespace()
	} else {
		gvk = oldObj.GetObjectKind().GroupVersionKind()
		name = oldObj.GetName()
		namespace = oldObj.GetNamespace()

		if newObj == nil {
			operation = admissionv1.Delete
		} else {
			operation = admissionv1.Update
		}
	}

	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: operation,
			Kind: metav1.GroupVersionKind{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    gvk.Kind,
			},
			Name:      name,
			Namespace: namespace,
			Object: runtime.RawExtension{
				Object: newObj,
			},
			OldObject: runtime.RawExtension{
				Object: oldObj,
			},
		},
	}
}
