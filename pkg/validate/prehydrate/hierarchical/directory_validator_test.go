package hierarchical

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

func TestObjectDirectoryValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "Role with unspecified namespace",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Namespace(""))},
							},
						},
					},
				},
			},
		},
		{
			name: "Role under valid directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Namespace("hello"))},
							},
						},
					},
				},
			},
		},
		{
			name: "Role under invalid directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml", core.Namespace("world"))},
							},
						},
					},
				},
			},
			wantErr: metadata.IllegalMetadataNamespaceDeclarationError(fake.Role(core.Namespace("world")), "hello"),
		},
	}

	for _, tc := range testCases {
		nv := ObjectDirectoryValidator()
		t.Run(tc.name, func(t *testing.T) {
			err := nv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got ObjectDirectoryValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestNamespaceDirectoryValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "Namespace under valid directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
							},
						},
					},
				},
			},
		},
		{
			name: "Namespace under invalid directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello", core.Name("world"))},
							},
						},
					},
				},
			},
			wantErr: metadata.InvalidNamespaceNameError(fake.Namespace("namespaces/hello", core.Name("world")), "hello"),
		},
		{
			name: "Namespace under top-level namespaces directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.Namespace("namespaces", core.Name("hello"))},
					},
				},
			},
			wantErr: metadata.IllegalTopLevelNamespaceError(fake.Namespace("namespaces", core.Name("hello"))),
		},
	}

	for _, tc := range testCases {
		nv := NamespaceDirectoryValidator()
		t.Run(tc.name, func(t *testing.T) {
			err := nv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got NamespaceDirectoryValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

var reservedDir = "namespaces/" + configmanagement.ControllerNamespace

func TestDirectoryNameValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "Valid directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/hello"),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/hello")},
								{FileObject: fake.RoleAtPath("namespaces/hello/role.yaml")},
							},
						},
					},
				},
			},
		},
		{
			name: "Invalid directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash("namespaces/..."),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace("namespaces/...")},
								{FileObject: fake.RoleAtPath("namespaces/.../role.yaml")},
							},
						},
					},
				},
			},
			wantErr: syntax.InvalidDirectoryNameError(cmpath.RelativeSlash("namespaces/...")),
		},
		{
			name: "Reserved directory",
			root: &parsed.TreeRoot{
				Tree: &ast.TreeNode{
					Relative: cmpath.RelativeSlash("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: cmpath.RelativeSlash(reservedDir),
							Type:     node.Namespace,
							Objects: []*ast.NamespaceObject{
								{FileObject: fake.Namespace(reservedDir)},
								{FileObject: fake.RoleAtPath(reservedDir + "/role.yaml")},
							},
						},
					},
				},
			},
			wantErr: syntax.ReservedDirectoryNameError(cmpath.RelativeSlash(reservedDir)),
		},
	}

	for _, tc := range testCases {
		dv := DirectoryNameValidator()
		t.Run(tc.name, func(t *testing.T) {
			err := dv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got DirectoryNameValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
