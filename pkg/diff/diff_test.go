package diff

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/policycontroller"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cli-utils/pkg/common"
)

const (
	testNs1 = "fake-namespace-1"
	testNs2 = "fake-namespace-2"
)

func TestDiffType(t *testing.T) {
	testCases := []struct {
		name     string
		scope    declared.Scope
		declared core.Object
		actual   core.Object
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
			name:  "actual + no declared, cannot manage: noop",
			scope: "shipping",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				difftest.ManagedByRoot),
			want: NoOp,
		},
		{
			name: "actual + no declared, prevent deletion: unmanage",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)),
			want: Unmanage,
		},
		{
			name: "actual + no declared, system Namespace: unmanage",
			actual: fake.NamespaceObject(metav1.NamespaceSystem,
				syncertest.ManagementEnabled),
			want: Unmanage,
		},
		{
			name: "actual + no declared, gatekeeper Namespace: unmanage",
			actual: fake.NamespaceObject(policycontroller.NamespaceSystem,
				syncertest.ManagementEnabled),
			want: Unmanage,
		},
		{
			name: "actual + no declared, managed: delete",
			actual: fake.RoleObject(syncertest.ManagementEnabled,
				core.Name(metav1.NamespaceSystem)),
			want: Delete,
		},
		{
			name:   "actual + no declared, invalid management: unmanage",
			actual: fake.RoleObject(syncertest.ManagementInvalid),
			want:   Unmanage,
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

			if d := cmp.Diff(tc.want, diff.Operation(scope)); d != "" {
				t.Fatal(d)
			}
		})
	}
}

func TestThreeWay(t *testing.T) {
	tcs := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		newDeclared []core.Object
		// the previously declared resources.
		previousDeclared []core.Object
		// The actual state of the resources.
		actual []core.Object
		// expected diff.
		want []Diff
	}{
		{
			name: "Update and Create - no previously declared",
			newDeclared: []core.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			actual: []core.Object{
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
			newDeclared: []core.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			previousDeclared: []core.Object{
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
			newDeclared: []core.Object{
				fake.NamespaceObject("namespace/" + testNs1),
				fake.NamespaceObject("namespace/" + testNs2),
			},
			previousDeclared: []core.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			actual: []core.Object{
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
			newDeclared: []core.Object{},
			actual: []core.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: nil,
		},
		{
			name: "Delete - no actual",
			newDeclared: []core.Object{
				fake.NamespaceObject("namespace/" + testNs1),
			},
			previousDeclared: []core.Object{
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
			newDeclared: []core.Object{
				fake.NamespaceObject("namespace/" + testNs1),
			},
			previousDeclared: []core.Object{
				fake.NamespaceObject("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.NamespaceObject("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			actual: []core.Object{
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
			newDeclared := make(map[core.ID]core.Object)
			previousDeclared := make(map[core.ID]core.Object)
			actual := make(map[core.ID]core.Object)

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
	decl := map[core.ID]core.Object{
		core.IDOf(obj): obj,
	}
	actual := map[core.ID]core.Object{
		core.IDOf(obj): Unknown(),
	}
	diffs := ThreeWay(decl, nil, actual)
	if len(diffs) != 0 {
		t.Errorf("Want empty diffs with unknown; got %v", diffs)
	}
}
