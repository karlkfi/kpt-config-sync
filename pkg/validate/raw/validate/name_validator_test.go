package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestName(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "object with name",
			obj:  fake.Deployment("/", core.Name("foo")),
		},
		{
			name:    "object with empty name fails",
			obj:     fake.Deployment("/", core.Name("")),
			wantErr: fake.Error(nonhierarchical.MissingObjectNameErrorCode),
		},
		{
			name:    "object with invalid name fails",
			obj:     fake.Deployment("/", core.Name("FOO:BAR")),
			wantErr: fake.Error(nonhierarchical.InvalidMetadataNameErrorCode),
		},
		{
			name: "object with valid name for RBAC",
			obj:  fake.Role(core.Name("FOO:BAR")),
		},
		{
			name:    "object with invalid name for RBAC fails",
			obj:     fake.Role(core.Name("FOO/BAR")),
			wantErr: fake.Error(nonhierarchical.InvalidMetadataNameErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Name(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got Name() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
