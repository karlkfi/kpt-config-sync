package diff

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/policycontroller"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNs1 = "fake-namespace-1"
	testNs2 = "fake-namespace-2"
)

func TestDiffType(t *testing.T) {
	testCases := []struct {
		name     string
		scope    declared.Scope
		declared client.Object
		actual   client.Object
		want     Operation
	}{
		// Declared + no actual paths.
		{
			name:     "declared + no actual, management enabled: create",
			declared: fake.RoleObject(syncertest.ManagementEnabled),
			want:     Create,
		},
		{
			name:     "declared + no actual, management disabled: no op",
			declared: fake.RoleObject(syncertest.ManagementDisabled),
			want:     NoOp,
		},
		{
			name:     "declared + no actual, no management: error",
			scope:    declared.RootReconciler,
			declared: fake.RoleObject(),
			want:     Error,
		},
		// Declared + actual paths.
		{
			name:     "declared + actual, management enabled, can manage: update",
			scope:    declared.RootReconciler,
			declared: fake.RoleObject(syncertest.ManagementEnabled),
			actual:   fake.RoleObject(syncertest.ManagementEnabled),
			want:     Update,
		},
		{
			name:  "declared + actual, management enabled, namespace reconciler / root-owned object: conflict",
			scope: "foo",
			declared: fake.RoleObject(syncertest.ManagementEnabled,
				core.Namespace("foo")),
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Namespace("foo"),
				difftest.ManagedByRoot),
			want: ManagementConflict,
		},
		{
			name:     "declared + actual, management disabled, can manage, with meta: unmanage",
			scope:    declared.RootReconciler,
			declared: fake.RoleObject(syncertest.ManagementDisabled),
			actual:   fake.RoleObject(syncertest.ManagementEnabled),
			want:     Unmanage,
		},
		{
			name:     "declared + actual, management disabled, can manage, no meta: no op",
			scope:    declared.RootReconciler,
			declared: fake.RoleObject(syncertest.ManagementDisabled),
			actual:   fake.RoleObject(),
			want:     NoOp,
		},
		{
			name:  "declared + actual, management disabled, namespace reconciler / root-owned object: no op",
			scope: "shipping",
			declared: fake.RoleObject(syncertest.ManagementDisabled,
				core.Namespace("shipping")),
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Namespace("shipping"),
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name: "declared + actual, management disabled, root reconciler / namespace-owned object: no op",
			declared: fake.RoleObject(syncertest.ManagementDisabled,
				core.Namespace("shipping")),
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Namespace("shipping"),
				difftest.ManagedBy("shipping")),
			want: NoOp,
		},
		{
			name: "declared + actual, management disabled, root reconciler / empty management annotation object: unmanage",
			declared: fake.RoleObject(syncertest.ManagementDisabled,
				core.Namespace("shipping")),
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Namespace("shipping"),
				difftest.ManagedBy("")),
			want: Unmanage,
		},
		{
			name:     "declared + actual, declared management invalid: error",
			scope:    declared.RootReconciler,
			declared: fake.RoleObject(syncertest.ManagementInvalid),
			actual:   fake.RoleObject(),
			want:     Error,
		},
		{
			name:     "declared + actual, actual management invalid: error",
			scope:    declared.RootReconciler,
			declared: fake.RoleObject(syncertest.ManagementEnabled),
			actual:   fake.RoleObject(syncertest.ManagementInvalid),
			want:     Update,
		},
		// IgnoreMutation path.
		{
			name: "prevent mutations",
			declared: fake.RoleObject(
				syncertest.ManagementEnabled,
				core.Annotation(metadata.LifecycleMutationAnnotation, metadata.IgnoreMutation),
				core.Annotation("foo", "bar"),
			),
			actual: fake.RoleObject(
				syncertest.ManagementEnabled,
				core.Annotation(metadata.LifecycleMutationAnnotation, metadata.IgnoreMutation),
				core.Annotation("foo", "qux"),
			),
			want: NoOp,
		},
		{
			name: "update if actual missing annotation",
			// The use case where the user has added the annotation to an object. We
			// need to update the object so the actual one has the annotation now.
			declared: fake.RoleObject(
				syncertest.ManagementEnabled,
				core.Annotation(metadata.LifecycleMutationAnnotation, metadata.IgnoreMutation),
				core.Annotation("foo", "bar"),
			),
			actual: fake.RoleObject(
				syncertest.ManagementEnabled,
				core.Annotation("foo", "qux"),
			),
			want: Update,
		},
		{
			name: "update if declared missing annotation",
			// This corresponds to the use case where the user has removed the
			// annotation, indicating they want us to begin updating the object again.
			//
			// There is an edge case where users manually annotate in-cluster objects,
			// which has no effect on our behavior; we only honor declared lifecycle
			// annotations.
			declared: fake.RoleObject(
				syncertest.ManagementEnabled,
				core.Annotation("foo", "bar"),
			),
			actual: fake.RoleObject(
				core.Annotation(metadata.LifecycleMutationAnnotation, metadata.IgnoreMutation),
				core.Annotation("foo", "qux"),
			),
			want: Update,
		},
		// Actual + no declared paths.
		{
			name:   "actual + no declared, no meta: no-op",
			scope:  declared.RootReconciler,
			actual: fake.RoleObject(),
			want:   NoOp,
		},
		{
			name:  "actual + no declared, owned: noop",
			scope: declared.RootReconciler,
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.OwnerReference([]metav1.OwnerReference{
					{},
				})),
			want: NoOp,
		},
		{
			name:  "actual + no declared, cannot manage (actual is managed by Config Sync): noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_default-name"),
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name:  "actual + no declared, cannot manage (actual is not managed by Config Sync): noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_wrong-name"),
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name:  "actual + no declared, not managed by Config Sync but has other Config Sync annotations: noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.TokenAnnotation,
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name:  "actual + no declared, not managed by Config Sync (the configsync.gke.io/resource-id annotation is unset): noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name:  "actual + no declared, not managed by Config Sync (the configmanagement.gke.io/managed annotation is set to disabled): noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.ManagementDisabled,
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name:  "actual + no declared, not managed by Config Sync (the configsync.gke.io/resource-id annotation is incorrect): noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_wrong-name"),
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name: "actual + no declared, prevent deletion: unmanage",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_default-name"),
				core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			want: Unmanage,
		},
		{
			name: "actual + no declared, system Namespace: unmanage",
			actual: fake.NamespaceObject(metav1.NamespaceSystem,
				core.Annotation(metadata.ResourceIDKey, "_namespace_kube-system"),
				syncertest.ManagementEnabled),
			want: Unmanage,
		},
		{
			name: "actual + no declared, gatekeeper Namespace: unmanage",
			actual: fake.NamespaceObject(policycontroller.NamespaceSystem,
				core.Annotation(metadata.ResourceIDKey, "_namespace_gatekeeper-system"),
				syncertest.ManagementEnabled),
			want: Unmanage,
		},
		{
			name: "actual + no declared, managed: delete",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_kube-system"),
				core.Name(metav1.NamespaceSystem)),
			want: Delete,
		},
		{
			name:   "actual + no declared, invalid management: unmanage",
			actual: fake.RoleObject(syncertest.ManagementInvalid),
			want:   NoOp,
		},
		// Error path.
		{
			name: "no declared or actual, no op (log error)",
			want: NoOp,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scope := tc.scope
			if scope == "" {
				scope = declared.RootReconciler
			}

			diff := Diff{
				Declared: tc.declared,
				Actual:   tc.actual,
			}

			ctx := context.Background()
			if d := cmp.Diff(tc.want, diff.Operation(ctx, scope)); d != "" {
				t.Fatal(d)
			}
		})
	}
}

func TestThreeWay(t *testing.T) {
	tcs := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		newDeclared []client.Object
		// the previously declared resources.
		previousDeclared []client.Object
		// The actual state of the resources.
		actual []client.Object
		// expected diff.
		want []Diff
	}{
		{
			name: "Update and Create - no previously declared",
			newDeclared: []client.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			actual: []client.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.NamespaceObject("namespace/" + testNs1),
					Actual:   fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
				},
				{
					Declared: fake.NamespaceObject("namespace/" + testNs2),
					Actual:   nil,
				},
			},
		},
		{
			name: "Update and Create - no actual",
			newDeclared: []client.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			previousDeclared: []client.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.NamespaceObject("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: fake.NamespaceObject("namespace/" + testNs2),
					Actual:   nil,
				},
			},
		},
		{
			name: "Update and Create - with previousDeclared and actual",
			newDeclared: []client.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			previousDeclared: []client.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			actual: []client.Object{
				fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.NamespaceObject("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: fake.NamespaceObject("namespace/" + testNs2),
					Actual:   fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
				},
			},
		},
		{
			name:        "Noop - with actual and no declared",
			newDeclared: []client.Object{},
			actual: []client.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: nil,
		},
		{
			name: "Delete - no actual",
			newDeclared: []client.Object{
				fake.NamespaceObject("namespace/" + testNs1),
			},
			previousDeclared: []client.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.NamespaceObject("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: nil,
					Actual:   fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
				},
			},
		},
		{
			name: "Delete - with previous declared and actual",
			newDeclared: []client.Object{
				fake.NamespaceObject("namespace/" + testNs1),
			},
			previousDeclared: []client.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			actual: []client.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			want: []Diff{
				{
					Declared: fake.NamespaceObject("namespace/" + testNs1),
					Actual:   fake.NamespaceObject("namespace/" + testNs1),
				},
				{
					Declared: nil,
					Actual:   fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			newDeclared := make(map[core.ID]client.Object)
			previousDeclared := make(map[core.ID]client.Object)
			actual := make(map[core.ID]client.Object)

			for _, d := range tc.newDeclared {
				newDeclared[core.IDOf(d)] = d
			}
			for _, pd := range tc.previousDeclared {
				previousDeclared[core.IDOf(pd)] = pd
			}
			for _, a := range tc.actual {
				actual[core.IDOf(a)] = a
			}

			diffs := ThreeWay(newDeclared, previousDeclared, actual)
			if diff := cmp.Diff(diffs, tc.want,
				cmpopts.SortSlices(func(x, y Diff) bool { return x.GetName() < y.GetName() })); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestUnknown(t *testing.T) {
	obj := fake.NamespaceObject("hello")
	decl := map[core.ID]client.Object{
		core.IDOf(obj): obj,
	}
	actual := map[core.ID]client.Object{
		core.IDOf(obj): Unknown(),
	}
	diffs := ThreeWay(decl, nil, actual)
	if len(diffs) != 0 {
		t.Errorf("Want empty diffs with unknown; got %v", diffs)
	}
}
