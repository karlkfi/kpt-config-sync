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
	legalLabel    = "supported"
	legalLabel2   = "supported2"
	illegalLabel  = v1.ConfigManagementPrefix + "unsupported"
	illegalLabel2 = v1alpha1.ConfigSyncPrefix + "unsupported2"
)

func TestLabelValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "no labels",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(),
					fake.RoleBinding(),
				},
			},
		},
		{
			name: "legal labels",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(core.Label(legalLabel, "a")),
					fake.RoleBinding(core.Label(legalLabel2, "a")),
				},
			},
		},
		{
			name: "illegal ConfigManagement label",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.Role(core.Label(illegalLabel, "a")),
				},
			},
			wantErr: metadata.IllegalLabelDefinitionError(fake.Role(), []string{illegalLabel}),
		},
		{
			name: "illegal ConfigSync label",
			root: &parsed.FlatRoot{
				NamespaceObjects: []ast.FileObject{
					fake.RoleBinding(core.Label(illegalLabel2, "a")),
				},
			},
			wantErr: metadata.IllegalLabelDefinitionError(fake.RoleBinding(), []string{illegalLabel2}),
		},
	}

	for _, tc := range testCases {
		lv := LabelValidator()
		t.Run(tc.name, func(t *testing.T) {

			err := lv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got AnnotationValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
