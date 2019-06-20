package filter

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGroup(t *testing.T) {
	testCases := []struct {
		name     string
		obj      ast.FileObject
		expected bool
	}{
		{
			name:     "foo matches foo",
			obj:      fake.Unstructured(fake.GVK(kinds.Role(), fake.Group("foo"))),
			expected: true,
		},
		{
			name: "foo does not match bar",
			obj:  fake.Unstructured(fake.GVK(kinds.Role(), fake.Group("bar"))),
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
			obj:      fake.Unstructured(fake.GVK(kinds.Role(), fake.Kind("Role"))),
			expected: true,
		},
		{
			name: "Role does not match RoleBinding",
			obj:  fake.Unstructured(fake.GVK(kinds.Role(), fake.Kind("RoleBinding"))),
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
			obj:      fake.Role(object.Namespace("prod")),
			expected: true,
		},
		{
			name: "prod does not match dev",
			obj:  fake.Role(object.Namespace("dev")),
		},
		{
			name:     "prod matches Namespace prod",
			obj:      fake.Namespace("namespaces/prod"),
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
			obj:      fake.Role(object.Name("admin")),
			expected: true,
		},
		{
			name:   "admin does not match user",
			filter: "admin",
			obj:    fake.Role(object.Name("user")),
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
			obj:      fake.Role(object.Name("prod:admin")),
			expected: true,
		},
		{
			name:  "prod does not match dev:admin",
			group: "prod",
			obj:   fake.Role(object.Name("dev:admin")),
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
			obj:      fake.Role(object.Label("version", "")),
			expected: true,
		},
		{
			name:  "version does not match instance",
			label: "version",
			obj:   fake.Role(object.Label("instance", "")),
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

func withOwner(controller *bool) object.MetaMutator {
	return func(object v1.Object) {
		owners := object.GetOwnerReferences()
		owners = append(owners, v1.OwnerReference{
			Controller: controller,
		})
		object.SetOwnerReferences(owners)
	}
}

func TestControlled(t *testing.T) {
	// Declared because you can't take the address of a constant.
	trueC := true
	falseC := false

	testCases := []struct {
		name     string
		obj      ast.FileObject
		expected bool
	}{
		{
			name: "no controller returns false",
			obj:  fake.Role(),
		},
		{
			name: "nil controller returns false",
			obj:  fake.Role(withOwner(nil)),
		},
		{
			name: "false controller returns false",
			obj:  fake.Role(withOwner(&falseC)),
		},
		{
			name:     "true controller returns true",
			obj:      fake.Role(withOwner(&trueC)),
			expected: true,
		},
		{
			name:     "false and true controller returns true",
			obj:      fake.Role(withOwner(&falseC), withOwner(&trueC)),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := Controlled()(tc.obj)

			if tc.expected != actual {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}
