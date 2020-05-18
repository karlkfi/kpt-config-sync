package differ

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func buildUnstructured(opts ...func(*unstructured.Unstructured)) *unstructured.Unstructured {
	result := &unstructured.Unstructured{}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

func name(s string) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		u.SetName(s)
	}
}

func managed(s string) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		core.SetAnnotation(u, v1.ResourceManagementKey, s)
	}
}

func owned() func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		owners := u.GetOwnerReferences()
		owners = append(owners, metav1.OwnerReference{})
		u.SetOwnerReferences(owners)
	}
}

func TestDiffType(t *testing.T) {
	testCases := []struct {
		name       string
		declared   *unstructured.Unstructured
		actual     *unstructured.Unstructured
		expectType Type
	}{
		{
			name:       "in repo, create",
			declared:   buildUnstructured(),
			expectType: Create,
		},
		{
			name:       "in repo only and unmanaged, noop",
			declared:   buildUnstructured(managed(v1.ResourceManagementDisabled)),
			expectType: NoOp,
		},
		{
			name:       "in repo only, management invalid error",
			declared:   buildUnstructured(managed("invalid")),
			expectType: Error,
		},
		{
			name:       "in repo only, management empty string error",
			declared:   buildUnstructured(managed("")),
			expectType: Error,
		},
		{
			name:       "in both, update",
			declared:   buildUnstructured(),
			actual:     buildUnstructured(),
			expectType: Update,
		},
		{
			name:       "in both and owned, update",
			declared:   buildUnstructured(),
			actual:     buildUnstructured(owned()),
			expectType: Update,
		},
		{
			name:       "in both, update even though cluster has invalid annotation",
			declared:   buildUnstructured(),
			actual:     buildUnstructured(managed("invalid")),
			expectType: Update,
		},
		{
			name:       "in both, management disabled unmanage",
			declared:   buildUnstructured(managed(v1.ResourceManagementDisabled)),
			actual:     buildUnstructured(managed(v1.ResourceManagementEnabled)),
			expectType: Unmanage,
		},
		{
			name:       "in both, management disabled noop",
			declared:   buildUnstructured(managed(v1.ResourceManagementDisabled)),
			actual:     buildUnstructured(),
			expectType: NoOp,
		},
		{
			name:       "delete",
			actual:     buildUnstructured(managed(v1.ResourceManagementEnabled)),
			expectType: Delete,
		},
		{
			name:       "in cluster only, unset noop",
			actual:     buildUnstructured(),
			expectType: NoOp,
		},
		{
			name:       "in cluster only, remove invalid empty string",
			actual:     buildUnstructured(managed("")),
			expectType: Unmanage,
		},
		{
			name:       "in cluster only, remove invalid annotation",
			actual:     buildUnstructured(managed("invalid")),
			expectType: Unmanage,
		},
		{
			name:       "in cluster only and owned, do nothing",
			actual:     buildUnstructured(managed(v1.ResourceManagementEnabled), owned()),
			expectType: NoOp,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff := Diff{
				Declared: tc.declared,
				Actual:   tc.actual,
			}

			if d := cmp.Diff(tc.expectType, diff.Type()); d != "" {
				t.Fatal(d)
			}
		})
	}
}

func TestMultipleDifftypes(t *testing.T) {
	testcases := []struct {
		name                string
		declared            []*unstructured.Unstructured
		actuals             []*unstructured.Unstructured
		allDeclaredVersions map[string]bool
		expect              map[string]*Diff
		expectTypes         map[string]Type
		expectPanic         bool
	}{
		{
			name:        "empty returns empty",
			expect:      map[string]*Diff{},
			expectTypes: map[string]Type{},
		},
		{
			name: "not declared and in actual and management enabled, but in different version returns no diff",
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed(v1.ResourceManagementEnabled)),
			},
			allDeclaredVersions: map[string]bool{"foo": true},
			expect:              map[string]*Diff{},
			expectTypes:         map[string]Type{},
		},
		{
			name: "multiple diff types works",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
				buildUnstructured(name("bar")),
				buildUnstructured(name("qux"), managed(v1.ResourceManagementDisabled)),
			},
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("bar"), managed(v1.ResourceManagementEnabled)),
				buildUnstructured(name("qux")),
				buildUnstructured(name("mun"), managed(v1.ResourceManagementEnabled)),
			},
			allDeclaredVersions: map[string]bool{
				"foo": true, "bar": true, "qux": true,
			},
			expect: map[string]*Diff{
				"foo": {
					Name:     "foo",
					Declared: buildUnstructured(name("foo")),
				},
				"bar": {
					Name:     "bar",
					Declared: buildUnstructured(name("bar")),
					Actual:   buildUnstructured(name("bar"), managed(v1.ResourceManagementEnabled)),
				},
				"qux": {
					Name:     "qux",
					Declared: buildUnstructured(name("qux"), managed(v1.ResourceManagementDisabled)),
					Actual:   buildUnstructured(name("qux")),
				},
				"mun": {
					Name:   "mun",
					Actual: buildUnstructured(name("mun"), managed(v1.ResourceManagementEnabled)),
				},
			},
			expectTypes: map[string]Type{
				"foo": Create,
				"bar": Update,
				"qux": NoOp,
				"mun": Delete,
			},
		},
		{
			name: "duplicate declarations panics",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
				buildUnstructured(name("foo")),
			},
			expectPanic: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if x := recover(); x != nil {
					if _, ok := x.(invalidInput); ok && tc.expectPanic {
						return
					}
					t.Fatal(x)
				}
			}()

			diffs := Diffs(tc.declared, tc.actuals, tc.allDeclaredVersions)

			if len(tc.declared) > 0 {
				fmt.Printf("%v\n", tc.declared[0].Object)
				fmt.Println("name: ", tc.declared[0].GetName())
			}

			diffsMap := make(map[string]*Diff)
			diffTypesMap := make(map[string]Type)
			for _, diff := range diffs {
				fmt.Println(diff)
				diffsMap[diff.Name] = diff
				diffTypesMap[diff.Name] = diff.Type()
			}

			if tDiff := cmp.Diff(tc.expect, diffsMap); tDiff != "" {
				t.Fatal(tDiff)
			}

			if tDiff := cmp.Diff(tc.expectTypes, diffTypesMap); tDiff != "" {
				t.Fatal(tDiff)
			}
		})
	}
}
