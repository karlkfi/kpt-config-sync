package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	csmetadata "github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	legalAnnotation = "supported"
	cmAnnotation    = csmetadata.ConfigManagementPrefix + "unsupported"
	csAnnotation    = configsync.ConfigSyncPrefix + "unsupported"
)

func TestAnnotations(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.MultiError
	}{
		{
			name: "no annotations",
			obj:  fake.Role(),
		},
		{
			name: "legal annotation",
			obj:  fake.Role(core.Annotation(legalAnnotation, "a")),
		},
		{
			name: "legal namespace selector annotation",
			obj:  fake.Role(core.Annotation(csmetadata.NamespaceSelectorAnnotationKey, "a")),
		},
		{
			name: "legal legacy cluster selector annotation",
			obj:  fake.Role(core.Annotation(csmetadata.LegacyClusterSelectorAnnotationKey, "a")),
		},
		{
			name: "legal inline cluster selector annotation",
			obj:  fake.Role(core.Annotation(csmetadata.ClusterNameSelectorAnnotationKey, "a")),
		},
		{
			name: "legal management annotation",
			obj:  fake.RoleBinding(core.Annotation(csmetadata.ResourceManagementKey, "a")),
		},
		{
			name:    "illegal ConfigManagement annotation",
			obj:     fake.Role(core.Annotation(cmAnnotation, "a")),
			wantErr: metadata.IllegalAnnotationDefinitionError(fake.Role(), []string{cmAnnotation}),
		},
		{
			name:    "illegal ConfigSync annotation",
			obj:     fake.RoleBinding(core.Annotation(csAnnotation, "a")),
			wantErr: metadata.IllegalAnnotationDefinitionError(fake.RoleBinding(), []string{csAnnotation}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Annotations(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got Annotations() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
