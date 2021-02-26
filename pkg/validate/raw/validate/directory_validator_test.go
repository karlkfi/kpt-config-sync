package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDirectory(t *testing.T) {
	testCases := []struct {
		name    string
		obj     ast.FileObject
		wantErr status.Error
	}{
		{
			name: "Role with unspecified namespace",
			obj:  fake.RoleAtPath("namespaces/hello/role.yaml", core.Namespace("")),
		},
		{
			name: "Role under valid directory",
			obj:  fake.RoleAtPath("namespaces/hello/role.yaml", core.Namespace("hello")),
		},
		{
			name:    "Role under invalid directory",
			obj:     fake.RoleAtPath("namespaces/hello/role.yaml", core.Namespace("world")),
			wantErr: metadata.IllegalMetadataNamespaceDeclarationError(fake.Role(core.Namespace("world")), "hello"),
		},
		{
			name: "Namespace under valid directory",
			obj:  fake.Namespace("namespaces/hello"),
		},
		{
			name:    "Namespace under invalid directory",
			obj:     fake.Namespace("namespaces/hello", core.Name("world")),
			wantErr: metadata.InvalidNamespaceNameError(fake.Namespace("namespaces/hello", core.Name("world")), "hello"),
		},
		{
			name:    "Namespace under top-level namespaces directory",
			obj:     fake.Namespace("namespaces", core.Name("hello")),
			wantErr: metadata.IllegalTopLevelNamespaceError(fake.Namespace("namespaces", core.Name("hello"))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Directory(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got Directory() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
