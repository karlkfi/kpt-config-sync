package validation

import (
	"testing"

	"github.com/google/nomos/bespin/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
)

func hierarchyNode() *ast.TreeNode {
	return &ast.TreeNode{}
}

func organizationNode() *ast.TreeNode {
	return &ast.TreeNode{
		Objects: []*ast.NamespaceObject{{FileObject: asttesting.NewFakeFileObject(kinds.Organization().WithVersion(""), "")}},
	}
}

func folderNode() *ast.TreeNode {
	return &ast.TreeNode{
		Objects: []*ast.NamespaceObject{{FileObject: asttesting.NewFakeFileObject(kinds.Folder().WithVersion(""), "")}},
	}
}

func projectNode() *ast.TreeNode {
	return &ast.TreeNode{
		Objects: []*ast.NamespaceObject{{FileObject: asttesting.NewFakeFileObject(kinds.Project().WithVersion(""), "")}},
	}
}

func TestMaxDepthValidatorVisitTreeNode(t *testing.T) {
	var testCases = []struct {
		name       string
		hierarchy  []*ast.TreeNode
		shouldFail bool
	}{
		{
			name:      "depth 0",
			hierarchy: []*ast.TreeNode{},
		},
		{
			name:      "depth 1",
			hierarchy: []*ast.TreeNode{folderNode()},
		},
		{
			name:      "depth 4",
			hierarchy: []*ast.TreeNode{folderNode(), folderNode(), folderNode(), folderNode()},
		},
		{
			name:       "depth 5",
			hierarchy:  []*ast.TreeNode{folderNode(), folderNode(), folderNode(), folderNode(), folderNode()},
			shouldFail: true,
		},
		{
			name:      "depth 4 project",
			hierarchy: []*ast.TreeNode{folderNode(), folderNode(), folderNode(), folderNode(), projectNode()},
		},
		{
			name:       "depth 5 project",
			hierarchy:  []*ast.TreeNode{folderNode(), folderNode(), folderNode(), folderNode(), folderNode(), projectNode()},
			shouldFail: true,
		},
		{
			name:      "org depth 0",
			hierarchy: []*ast.TreeNode{organizationNode()},
		},
		{
			name:      "org depth 1",
			hierarchy: []*ast.TreeNode{organizationNode(), folderNode()},
		},
		{
			name:      "org depth 4",
			hierarchy: []*ast.TreeNode{organizationNode(), folderNode(), folderNode(), folderNode(), folderNode()},
		},
		{
			name:       "org depth 5",
			hierarchy:  []*ast.TreeNode{organizationNode(), folderNode(), folderNode(), folderNode(), folderNode(), folderNode()},
			shouldFail: true,
		},
		{
			name:      "org depth 4 project",
			hierarchy: []*ast.TreeNode{organizationNode(), folderNode(), folderNode(), folderNode(), folderNode(), projectNode()},
		},
		{
			name:       "org depth 5 project",
			hierarchy:  []*ast.TreeNode{organizationNode(), folderNode(), folderNode(), folderNode(), folderNode(), folderNode(), projectNode()},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := &ast.Root{
				Tree: hierarchyNode(),
			}
			curNode := root.Tree
			for _, node := range tc.hierarchy {
				curNode.Children = append(curNode.Children, node)
				curNode = node
			}

			v := NewMaxFolderDepthValidator()
			root.Accept(v)

			if tc.shouldFail {
				vettesting.ExpectErrors([]string{vet.UndocumentedErrorCode}, v.Error(), t)
			} else {
				vettesting.ExpectErrors(nil, v.Error(), t)
			}
		})
	}
}
