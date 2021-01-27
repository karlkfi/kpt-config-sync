package common

import (
	"errors"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

const (
	legalAnnotation    = "supported"
	legalAnnotation2   = "supported2"
	illegalAnnotation  = v1.ConfigManagementPrefix + "unsupported"
	illegalAnnotation2 = v1alpha1.ConfigSyncPrefix + "unsupported2"
)

func TestAnnotationValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "no annotations",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(),
					fake.RoleBinding(),
				},
			},
		},
		{
			name: "legal annotations",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(core.Annotation(legalAnnotation, "a")),
					fake.RoleBinding(core.Annotation(legalAnnotation2, "a")),
				},
			},
		},
		{
			name: "legal source annotations",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(core.Annotation(v1.NamespaceSelectorAnnotationKey, "a")),
					fake.Role(core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "a")),
					fake.RoleBinding(core.Annotation(v1.ResourceManagementKey, "a")),
					fake.RoleBinding(core.Annotation(v1alpha1.ClusterNameSelectorAnnotationKey, "a")),
				},
			},
		},
		{
			name: "illegal ConfigManagement annotation",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(core.Annotation(illegalAnnotation, "a")),
				},
			},
			wantErr: metadata.IllegalAnnotationDefinitionError(fake.Role(), []string{illegalAnnotation}),
		},
		{
			name: "illegal ConfigSync annotation",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.RoleBinding(core.Annotation(illegalAnnotation2, "a")),
				},
			},
			wantErr: metadata.IllegalAnnotationDefinitionError(fake.RoleBinding(), []string{illegalAnnotation2}),
		},
	}

	for _, tc := range testCases {
		av := AnnotationValidator()
		t.Run(tc.name, func(t *testing.T) {

			err := av(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got AnnotationValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
