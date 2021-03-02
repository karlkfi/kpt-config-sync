package validate

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
)

func TestValidManagementAnnotation(t *testing.T) {
	testCases := []struct {
		name string
		obj  ast.FileObject
		want status.Error
	}{
		{
			name: "no management annotation",
			obj:  fake.Role(),
		},
		{
			name: "disabled management passes",
			obj:  fake.Role(syncertest.ManagementDisabled),
		},
		{
			name: "enabled management fails",
			obj:  fake.Role(syncertest.ManagementEnabled),
			want: fake.Error(nonhierarchical.IllegalManagementAnnotationErrorCode),
		},
		{
			name: "invalid management fails",
			obj:  fake.Role(core.Annotation(v1.ResourceManagementKey, "invalid")),
			want: fake.Error(nonhierarchical.IllegalManagementAnnotationErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := nonhierarchical.ValidManagementAnnotation(tc.obj)
			if !errors.Is(err, tc.want) {
				t.Errorf("got ValidateCRDName() error %v, want %v", err, tc.want)
			}
		})
	}
}
