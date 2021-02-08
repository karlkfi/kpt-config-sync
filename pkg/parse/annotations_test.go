package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestAddAnnotationsAndLabels(t *testing.T) {
	testcases := []struct {
		name       string
		actual     []ast.FileObject
		expected   []ast.FileObject
		gc         gitContext
		commitHash string
	}{
		{
			name:     "empty list",
			actual:   []ast.FileObject{},
			expected: []ast.FileObject{},
		},
		{
			name: "nil annotation without env",
			gc: gitContext{
				Repo:   "git@github.com/foo",
				Branch: "main",
				Rev:    "HEAD",
			},
			commitHash: "1234567",
			actual:     []ast.FileObject{fake.Role()},
			expected: []ast.FileObject{fake.Role(
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, "enabled"),
				core.Annotation(v1alpha1.ResourceManagerKey, "some-namespace"),
				core.Annotation(v1.SyncTokenAnnotationKey, "1234567"),
				core.Annotation(v1alpha1.GitContextKey, `{"repo":"git@github.com/foo","branch":"main","rev":"HEAD"}`),
				core.Annotation(kptapplier.OwningInventoryKey, kptapplier.InventoryID("some-namespace")),
			)},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if err := addAnnotationsAndLabels(tc.actual, "some-namespace", tc.gc, tc.commitHash); err != nil {
				t.Fatalf("Failed to add annotations and labels: %v", err)
			}
			if diff := cmp.Diff(tc.expected, tc.actual, ast.CompareFileObject); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}
