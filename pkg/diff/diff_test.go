package diff

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNs1 = "fake-namespace-1"
	testNs2 = "fake-namespace-2"
)

func TestDiffType(t *testing.T) {
	testCases := []struct {
		name       string
		scope      declared.Scope
		declared   core.Object
		actual     core.Object
		expectType Type
	}{
		{
			name:       "in repo, create",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementEnabled),
			expectType: Create,
		},
		{
			name:       "in repo only and unmanaged, noop",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementDisabled),
			expectType: NoOp,
		},
		{
			name:       "in repo only, management invalid error",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "invalid")),
			expectType: Error,
		},
		{
			name:       "in repo only, management empty string error",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "")),
			expectType: Error,
		},
		{
			name:       "in both, update",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementEnabled),
			actual:     fake.ClusterRoleObject(syncertest.ManagementEnabled),
			expectType: Update,
		},
		{
			name:       "in both and owned, update",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementEnabled),
			actual:     fake.ClusterRoleObject(core.OwnerReference([]metav1.OwnerReference{{}})),
			expectType: Update,
		},
		{
			name:       "in both, update even though cluster has invalid annotation",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementEnabled),
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "invalid")),
			expectType: Update,
		},
		{
			name:       "in both, management disabled unmanage",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementDisabled),
			actual:     fake.ClusterRoleObject(syncertest.ManagementEnabled),
			expectType: Unmanage,
		},
		{
			name:       "in both, management disabled noop",
			scope:      declared.RootReconciler,
			declared:   fake.ClusterRoleObject(syncertest.ManagementDisabled),
			actual:     fake.ClusterRoleObject(),
			expectType: NoOp,
		},
		{
			name:       "delete",
			scope:      declared.RootReconciler,
			actual:     fake.ClusterRoleObject(syncertest.ManagementEnabled),
			expectType: Delete,
		},
		{
			name:       "in cluster only, unset noop",
			scope:      declared.RootReconciler,
			actual:     fake.ClusterRoleObject(),
			expectType: NoOp,
		},
		{
			name:       "in cluster only, remove invalid empty string",
			scope:      declared.RootReconciler,
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "")),
			expectType: Unmanage,
		},
		{
			name:       "in cluster only, remove invalid annotation",
			scope:      declared.RootReconciler,
			actual:     fake.ClusterRoleObject(core.Annotation(v1.ResourceManagementKey, "invalid")),
			expectType: Unmanage,
		},
		{
			name:  "in cluster only and owned, do nothing",
			scope: declared.RootReconciler,
			actual: fake.ClusterRoleObject(syncertest.ManagementEnabled,
				core.OwnerReference([]metav1.OwnerReference{{}})),
			expectType: NoOp,
		},
		{
			name:  "in cluster only and owned and prevent deletion, unmanage",
			scope: declared.RootReconciler,
			actual: fake.ClusterRoleObject(syncertest.ManagementEnabled,
				core.OwnerReference([]metav1.OwnerReference{{}}),
				core.Annotation(lifecycle.Deletion, lifecycle.PreventDeletion)),
			expectType: NoOp,
		},
		{
			name:       "in-namespace-repo object managed by correct Namespace reconciler",
			scope:      "shipping",
			declared:   fake.RoleObject(syncertest.ManagementEnabled),
			actual:     fake.RoleObject(syncertest.ManagementEnabled, difftest.ManagedBy("shipping")),
			expectType: Update,
		},
		{
			// Namespace cannot take over ownership from Root.
			name:       "in-namespace-repo object managed by Root reconciler",
			scope:      "shipping",
			declared:   fake.RoleObject(syncertest.ManagementEnabled),
			actual:     fake.RoleObject(syncertest.ManagementEnabled, difftest.ManagedBy(declared.RootReconciler)),
			expectType: ManagementConflict,
		},
		// Root always wins.
		{
			// Root will take over ownership from Namespace.
			name:       "in-root-repo object managed by Namespace reconciler",
			scope:      declared.RootReconciler,
			declared:   fake.RoleObject(syncertest.ManagementEnabled, difftest.ManagedBy(declared.RootReconciler)),
			actual:     fake.RoleObject(syncertest.ManagementEnabled, difftest.ManagedBy("shipping")),
			expectType: Update,
		},
		{
			name:       "in-root-repo object managed by Root reconciler",
			scope:      declared.RootReconciler,
			declared:   fake.RoleObject(syncertest.ManagementEnabled, difftest.ManagedBy(declared.RootReconciler)),
			actual:     fake.RoleObject(syncertest.ManagementEnabled, difftest.ManagedBy(declared.RootReconciler)),
			expectType: Update,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff := Diff{
				Declared: tc.declared,
				Actual:   tc.actual,
			}

			if d := cmp.Diff(tc.expectType, diff.Type(tc.scope)); d != "" {
				t.Fatal(d)
			}
		})
	}
}

func TestThreeWay(t *testing.T) {
	tcs := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		newDeclared []ast.FileObject
		// the previously declared resources.
		previousDeclared []ast.FileObject
		// The actual state of the resources.
		actual []ast.FileObject
		// expected diff.
		want []Diff
	}{
		{
			name: "Update and Create - no previously declared",
			newDeclared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				fake.Namespace("namespace/" + testNs2),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
				},
				{
					Declared: fake.Namespace("namespace/" + testNs2),
					Actual:   nil,
				},
			},
		},
		{
			name: "Update and Create - no actual",
			newDeclared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				fake.Namespace("namespace/" + testNs2),
			},
			previousDeclared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: fake.Namespace("namespace/" + testNs2),
					Actual:   nil,
				},
			},
		},
		{
			name: "Update and Create - with previousDeclared and actual",
			newDeclared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				fake.Namespace("namespace/" + testNs2),
			},
			previousDeclared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: fake.Namespace("namespace/" + testNs2),
					Actual:   fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
				},
			},
		},
		{
			name:        "Noop - with actual and no declared",
			newDeclared: []ast.FileObject{},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: nil,
		},
		{
			name: "Delete - no actual",
			newDeclared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			previousDeclared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: nil,
					Actual:   fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
				},
			},
		},
		{
			name: "Delete - with previous declared and actual",
			newDeclared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			previousDeclared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				fake.Namespace("namespace/" + testNs2),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   fake.Namespace("namespace/" + testNs1),
				},
				{
					Declared: nil,
					Actual:   fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
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

func TestTwoWay(t *testing.T) {
	tcs := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		declared []ast.FileObject
		// The actual state of the resources.
		actual []ast.FileObject
		// expected diff.
		want []Diff
	}{
		{
			name: "No Diff when declared is nil",
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: nil,
		},
		{
			name: "Diff - no actual",
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				fake.Namespace("namespace/" + testNs2),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: fake.Namespace("namespace/" + testNs2),
					Actual:   nil,
				},
			},
		},
		{
			name: "Diff for update- with previousDeclared and actual",
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				fake.Namespace("namespace/" + testNs2),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Diff{
				{
					Declared: fake.Namespace("namespace/" + testNs1),
					Actual:   nil,
				},
				{
					Declared: fake.Namespace("namespace/" + testNs2),
					Actual:   fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			declared := make(map[core.ID]core.Object)
			actual := make(map[core.ID]core.Object)

			for _, d := range tc.declared {
				declared[core.IDOf(d)] = d
			}
			for _, a := range tc.actual {
				actual[core.IDOf(a)] = a
			}

			diffs := TwoWay(declared, actual)
			if diff := cmp.Diff(diffs, tc.want,
				cmpopts.SortSlices(func(x, y Diff) bool { return x.GetName() < y.GetName() })); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}
