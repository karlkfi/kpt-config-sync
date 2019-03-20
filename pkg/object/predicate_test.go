package object_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/filter"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestPredicate(t *testing.T) {
	testCases := []struct {
		name      string
		predicate object.Predicate
		expected  bool
	}{
		{
			name:      "True returns true",
			predicate: object.True(),
			expected:  true,
		},
		{
			name:      "False returns false",
			predicate: object.False(),
		},
		{
			name:      "empty All returns true",
			predicate: object.All(),
			expected:  true,
		},
		{
			name:      "True And False returns false",
			predicate: object.All(object.True(), object.False()),
		},
		{
			name:      "False And True returns false",
			predicate: object.All(object.False(), object.True()),
		},
		{
			name:      "False And false returns false",
			predicate: object.All(object.False(), object.False()),
		},
		{
			name:      "empty Any returns false",
			predicate: object.Any(),
		},
		{
			name:      "True Or True returns true",
			predicate: object.Any(object.True(), object.True()),
			expected:  true,
		},
		{
			name:      "True Or False returns true",
			predicate: object.Any(object.True(), object.False()),
			expected:  true,
		},
		{
			name:      "False Or True returns true",
			predicate: object.Any(object.False(), object.True()),
			expected:  true,
		},
		{
			name:      "False Or False returns false",
			predicate: object.Any(object.False(), object.False()),
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
	isRole := filter.GroupKind(kinds.Role().GroupKind())
	isRoleBinding := filter.GroupKind(kinds.RoleBinding().GroupKind())

	role := fake.Build(kinds.Role(), object.Name("admin"))
	roleBinding := fake.Build(kinds.RoleBinding(), object.Name("admin"))

	objects := []ast.FileObject{role, roleBinding}

	testCases := []struct {
		name      string
		predicate object.Predicate
		expected  []ast.FileObject
	}{
		{
			name:      "False() returns both",
			predicate: object.False(),
			expected:  objects,
		},
		{
			name:      "True() returns neither",
			predicate: object.True(),
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
			predicate: object.All(isRole, isRoleBinding),
			expected:  objects,
		},
		{
			name:      "filter if is role OR is roleBinding returns none",
			predicate: object.Any(isRole, isRoleBinding),
			expected:  nil,
		},
		{
			name:      "filter out has name admin returns both",
			predicate: filter.Name("admin"),
			expected:  nil,
		},
		{
			name:      "has name admin + is role returns role",
			predicate: object.All(filter.Name("admin"), isRole),
			expected:  []ast.FileObject{roleBinding},
		},
		{
			name:      "has name admin + is rolebinding returns rolebinding",
			predicate: object.All(filter.Name("admin"), isRoleBinding),
			expected:  []ast.FileObject{role},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := object.Filter(objects, tc.predicate)

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
