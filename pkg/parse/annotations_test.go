package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/metadata"
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
			actual:     []ast.FileObject{fake.Role(core.Namespace("foo"))},
			expected: []ast.FileObject{fake.Role(
				core.Namespace("foo"),
				core.Label(metadata.ManagedByKey, metadata.ManagedByValue),
				core.Annotation(metadata.ResourceManagementKey, "enabled"),
				core.Annotation(metadata.ResourceManagerKey, "some-namespace"),
				core.Annotation(metadata.SyncTokenAnnotationKey, "1234567"),
				core.Annotation(metadata.GitContextKey, `{"repo":"git@github.com/foo","branch":"main","rev":"HEAD"}`),
				core.Annotation(metadata.OwningInventoryKey, applier.InventoryID("some-namespace")),
				core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_foo_default-name"),
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
