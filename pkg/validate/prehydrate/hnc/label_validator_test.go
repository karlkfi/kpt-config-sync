package hnc

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	oldhnc "github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/validate/parsed"
)

const (
	legalLabel            = "label"
	illegalSuffixedLabel  = "unsupported" + oldhnc.DepthSuffix
	illegalSuffixedLabel2 = "unsupported2" + oldhnc.DepthSuffix
)

func treeRootWith(obj ast.FileObject) *parsed.TreeRoot {
	return &parsed.TreeRoot{
		Tree: &ast.TreeNode{
			Relative: cmpath.RelativeSlash("namespaces"),
			Type:     node.AbstractNamespace,
			Children: []*ast.TreeNode{
				{
					Relative: cmpath.RelativeSlash("namespaces/hello"),
					Type:     node.Namespace,
					Objects: []*ast.NamespaceObject{
						{FileObject: fake.Namespace("namespaces/hello")},
						{FileObject: obj},
					},
				},
			},
		},
	}
}

func TestDepthLabelValidator(t *testing.T) {
	testCases := []struct {
		name    string
		root    parsed.Root
		wantErr status.MultiError
	}{
		{
			name: "no labels",
			root: treeRootWith(fake.RoleAtPath("namespaces/hello/role.yaml")),
		},
		{
			name: "one legal label",
			root: treeRootWith(fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(legalLabel, ""))),
		},
		{
			name: "one illegal label",
			root: treeRootWith(fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(illegalSuffixedLabel, ""))),
			wantErr: oldhnc.IllegalDepthLabelError(fake.Role(), []string{illegalSuffixedLabel}),
		},
		{
			name: "two illegal labels",
			root: treeRootWith(fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(illegalSuffixedLabel, ""),
				core.Label(illegalSuffixedLabel2, ""))),
			wantErr: oldhnc.IllegalDepthLabelError(fake.Role(), []string{illegalSuffixedLabel, illegalSuffixedLabel2}),
		},
		{
			name: "one legal and one illegal label",
			root: treeRootWith(fake.RoleAtPath("namespaces/hello/role.yaml",
				core.Label(legalLabel, ""),
				core.Label(illegalSuffixedLabel, ""))),
			wantErr: oldhnc.IllegalDepthLabelError(fake.Role(), []string{illegalSuffixedLabel}),
		},
	}

	for _, tc := range testCases {
		dv := DepthLabelValidator()
		t.Run(tc.name, func(t *testing.T) {
			err := dv(tc.root)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got DepthLabelValidator() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
