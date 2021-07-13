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
	legalLabel = "supported"
	cmLabel    = csmetadata.ConfigManagementPrefix + "unsupported"
	csLabel    = configsync.ConfigSyncPrefix + "unsupported2"
)

func TestLabels(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.MultiError
	}{
		{
			name: "no labels",
			obj:  fake.Role(),
		},
		{
			name: "legal label",
			obj:  fake.Role(core.Label(legalLabel, "a")),
		},
		{
			name:    "illegal ConfigManagement label",
			obj:     fake.Role(core.Label(cmLabel, "a")),
			wantErr: metadata.IllegalLabelDefinitionError(fake.Role(), []string{cmLabel}),
		},
		{
			name:    "illegal ConfigSync label",
			obj:     fake.RoleBinding(core.Label(csLabel, "a")),
			wantErr: metadata.IllegalLabelDefinitionError(fake.RoleBinding(), []string{csLabel}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Labels(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got Labels() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
