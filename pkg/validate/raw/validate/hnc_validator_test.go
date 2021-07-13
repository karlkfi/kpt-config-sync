package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	illegalSuffixedLabel  = "unsupported" + metadata.DepthSuffix
	illegalSuffixedLabel2 = "unsupported2" + metadata.DepthSuffix
)

func TestHNCLabels(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "no labels",
			obj:  fake.RoleAtPath("namespaces/hello/role.yaml"),
		},
		{
			name: "one legal label",
			obj: fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(legalLabel, "")),
		},
		{
			name: "one illegal label",
			obj: fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(illegalSuffixedLabel, "")),
			wantErr: hnc.IllegalDepthLabelError(fake.Role(), []string{illegalSuffixedLabel}),
		},
		{
			name: "two illegal labels",
			obj: fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(illegalSuffixedLabel, ""),
				core.Label(illegalSuffixedLabel2, "")),
			wantErr: hnc.IllegalDepthLabelError(fake.Role(), []string{illegalSuffixedLabel, illegalSuffixedLabel2}),
		},
		{
			name: "one legal and one illegal label",
			obj: fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(legalLabel, ""),
				core.Label(illegalSuffixedLabel, "")),
			wantErr: hnc.IllegalDepthLabelError(fake.Role(), []string{illegalSuffixedLabel}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := HNCLabels(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got HNCLabels() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
