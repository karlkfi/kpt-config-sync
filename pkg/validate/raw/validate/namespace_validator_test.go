package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespace(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Role with unspecified namespace",
			obj:  fake.Role(core.Namespace("")),
		},
		{
			name: "Role with valid namespace",
			obj:  fake.Role(core.Namespace("hello")),
		},
		{
			name:    "Role with invalid namespace",
			obj:     fake.Role(core.Namespace("..invalid..")),
			wantErr: nonhierarchical.InvalidNamespaceError(fake.Role()),
		},
		{
			name:    "Role with illegal namespace",
			obj:     fake.Role(core.Namespace(configmanagement.ControllerNamespace)),
			wantErr: nonhierarchical.ObjectInIllegalNamespace(fake.Role()),
		},
		{
			name: "Valid namespace",
			obj:  fake.Namespace("hello"),
		},
		{
			name:    "Illegal namespace",
			obj:     fake.Namespace(configmanagement.ControllerNamespace),
			wantErr: nonhierarchical.IllegalNamespace(fake.Namespace(configmanagement.ControllerNamespace)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Namespace(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got Namespace() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
