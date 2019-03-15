package filter

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/testing/object"
)

func TestPredicate(t *testing.T) {
	testCases := []struct {
		name      string
		predicate Predicate
		expected  bool
	}{
		{
			name:      "True returns true",
			predicate: True(),
			expected:  true,
		},
		{
			name:      "False returns false",
			predicate: False(),
		},
		{
			name:      "empty All returns true",
			predicate: All(),
			expected:  true,
		},
		{
			name:      "True And False returns false",
			predicate: All(True(), False()),
		},
		{
			name:      "False And True returns false",
			predicate: All(False(), True()),
		},
		{
			name:      "False And false returns false",
			predicate: All(False(), False()),
		},
		{
			name:      "empty Any returns false",
			predicate: Any(),
		},
		{
			name:      "True Or True returns true",
			predicate: Any(True(), True()),
			expected:  true,
		},
		{
			name:      "True Or False returns true",
			predicate: Any(True(), False()),
			expected:  true,
		},
		{
			name:      "False Or True returns true",
			predicate: Any(False(), True()),
			expected:  true,
		},
		{
			name:      "False Or False returns false",
			predicate: Any(False(), False()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.predicate(ast.FileObject{})

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestPredicateObjects(t *testing.T) {
	isRole := GroupKind(kinds.Role().GroupKind())
	isRoleBinding := GroupKind(kinds.RoleBinding().GroupKind())

	role := object.Build(kinds.Role(), object.Name("admin"))
	roleBinding := object.Build(kinds.RoleBinding(), object.Name("admin"))

	objects := []ast.FileObject{role, roleBinding}

	testCases := []struct {
		name      string
		predicate Predicate
		expected  []ast.FileObject
	}{
		{
			name:      "False() returns both",
			predicate: False(),
			expected:  objects,
		},
		{
			name:      "True() returns neither",
			predicate: True(),
		},
		{
			name:      "filter out role only returns role",
			predicate: isRole,
			expected:  []ast.FileObject{roleBinding},
		},
		{
			name:      "filter out rolebinding only returns rolebinding",
			predicate: isRoleBinding,
			expected:  []ast.FileObject{role},
		},
		{
			name:      "filter out role + is rolebinding returns both",
			predicate: All(isRole, isRoleBinding),
			expected:  objects,
		},
		{
			name:      "filter if is role OR is roleBinding returns none",
			predicate: Any(isRole, isRoleBinding),
			expected:  nil,
		},
		{
			name:      "filter out has name admin returns both",
			predicate: Name("admin"),
			expected:  nil,
		},
		{
			name:      "has name admin + is role returns role",
			predicate: All(Name("admin"), isRole),
			expected:  []ast.FileObject{roleBinding},
		},
		{
			name:      "has name admin + is rolebinding returns rolebinding",
			predicate: All(Name("admin"), isRoleBinding),
			expected:  []ast.FileObject{role},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Objects(objects, tc.predicate)

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
