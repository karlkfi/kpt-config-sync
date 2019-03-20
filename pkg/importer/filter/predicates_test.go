package filter

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestGroup(t *testing.T) {
	testCases := []struct {
		name     string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "foo matches foo",
			obj:      fake.Build(fake.GVK(kinds.Role(), fake.Group("foo"))),
			expected: true,
		},
		{
			name: "foo does not match bar",
			obj:  fake.Build(fake.GVK(kinds.Role(), fake.Group("bar"))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Group("foo")(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestKind(t *testing.T) {
	testCases := []struct {
		name     string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "Role matches Role",
			obj:      fake.Build(fake.GVK(kinds.Role(), fake.Kind("Role"))),
			expected: true,
		},
		{
			name: "Role does not match RoleBinding",
			obj:  fake.Build(fake.GVK(kinds.Role(), fake.Kind("RoleBinding"))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Kind("Role")(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestNamespace(t *testing.T) {
	testCases := []struct {
		name     string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "prod matches prod",
			obj:      fake.Build(kinds.Role(), object.Namespace("prod")),
			expected: true,
		},
		{
			name: "prod does not match dev",
			obj:  fake.Build(kinds.Role(), object.Namespace("dev")),
		},
		{
			name:     "prod matches Namespace prod",
			obj:      fake.Build(kinds.Namespace(), object.Name("prod")),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Namespace("prod")(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestName(t *testing.T) {
	testCases := []struct {
		name     string
		filter   string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "admin matches admin",
			filter:   "admin",
			obj:      fake.Build(kinds.Role(), object.Name("admin")),
			expected: true,
		},
		{
			name:   "admin does not match user",
			filter: "admin",
			obj:    fake.Build(kinds.Role(), object.Name("user")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Name(tc.filter)(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestNameGroup(t *testing.T) {
	testCases := []struct {
		name     string
		group    string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "prod matches prod:admin",
			group:    "prod",
			obj:      fake.Build(kinds.Role(), object.Name("prod:admin")),
			expected: true,
		},
		{
			name:  "prod does not match dev:admin",
			group: "prod",
			obj:   fake.Build(kinds.Role(), object.Name("dev:admin")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := NameGroup(tc.group)(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestLabel(t *testing.T) {
	testCases := []struct {
		name     string
		label    string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "version matches version",
			label:    "version",
			obj:      fake.Build(kinds.Role(), object.Label("version", "")),
			expected: true,
		},
		{
			name:  "version does not match instance",
			label: "version",
			obj:   fake.Build(kinds.Role(), object.Label("instance", "")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Label(tc.label)(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}
