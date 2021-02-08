package selectors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestResolveFlatNamespaceSelectors(t *testing.T) {
	namespaceSelectorObject := fake.NamespaceSelectorObject(
		core.Name("dev-only"),
	)
	namespaceSelectorObject.Spec.Selector.MatchLabels = map[string]string{
		"environment": "dev",
	}
	namespaceSelector := fake.FileObject(namespaceSelectorObject, "prod-only-nss.yaml")

	testCases := []struct {
		name        string
		before      []ast.FileObject
		expected    []ast.FileObject
		expectError bool
	}{
		{
			name: "empty does nothing",
		},
		{
			name: "does nothing if no namespace-selector annotations",
			before: []ast.FileObject{
				namespaceSelector,
				fake.Namespace("namespaces/prod",
					core.Label("environment", "prod"),
				),
				fake.Role(core.Namespace("prod")),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/prod",
					core.Label("environment", "prod"),
				),
				fake.Role(core.Namespace("prod")),
			},
		},
		{
			name: "removes object with namespace-selector",
			before: []ast.FileObject{
				namespaceSelector,
				fake.Namespace("namespaces/prod",
					core.Label("environment", "prod"),
				),
				fake.Role(core.Namespace("prod"),
					core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only"),
				),
			},
			expected: []ast.FileObject{
				fake.Namespace("namespaces/prod",
					core.Label("environment", "prod"),
				),
			},
		},
		{
			name: "error on missing namespace-selector",
			before: []ast.FileObject{
				fake.Namespace("namespaces/prod",
					core.Label("environment", "prod"),
				),
				fake.Role(core.Namespace("prod"),
					core.Annotation(v1.NamespaceSelectorAnnotationKey, "dev-only"),
				),
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, errs := ResolveFlatNamespaceSelectors(tc.before)

			if tc.expectError || errs != nil {
				if tc.expectError && errs == nil {
					t.Fatal("expected error")
				}
				if !tc.expectError && errs != nil {
					t.Fatal("unexpected error", errs)
				}
				return
			}

			if diff := cmp.Diff(tc.expected, actual, ast.CompareFileObject); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
