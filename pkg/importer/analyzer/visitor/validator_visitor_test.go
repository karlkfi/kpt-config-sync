package visitor

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func testError() status.MultiError {
	return status.UndocumentedError("error")
}

func fakeObject() ast.FileObject {
	return fake.UnstructuredAtPath(kinds.Role(), "")
}

func failRoot(_ *ast.Root) status.MultiError {
	return testError()
}

func failSystemObject(_ *ast.SystemObject) status.MultiError {
	return testError()
}

func failClusterObject(_ *ast.ClusterObject) status.MultiError {
	return testError()
}

func failTreeNode(_ *ast.TreeNode) status.MultiError {
	return testError()
}

func failLeafTreeNode(n *ast.TreeNode) status.MultiError {
	if len(n.Children) == 0 {
		return testError()
	}
	return nil
}

func TestValidators(t *testing.T) {
	testCases := []struct {
		name        string
		root        ast.Root
		validator   *ValidatorVisitor
		failMessage string
	}{
		{
			name:        "rootValidator",
			root:        ast.Root{},
			failMessage: "does not visit root",
			validator:   NewRootValidator(failRoot),
		},
		{
			name:        "systemObjectValidator",
			root:        ast.Root{SystemObjects: []*ast.SystemObject{{FileObject: fakeObject()}}},
			failMessage: "does not visit System objects",
			validator:   NewSystemObjectValidator(failSystemObject),
		},
		{
			name:        "clusterObjectValidator",
			root:        ast.Root{ClusterObjects: []*ast.ClusterObject{{FileObject: fakeObject()}}},
			failMessage: "does not visit Cluster objects",
			validator:   NewClusterObjectValidator(failClusterObject),
		},
		{
			name:        "treeNodeValidator root",
			root:        ast.Root{Tree: &ast.TreeNode{}},
			failMessage: "does not visit root TreeNodes",
			validator:   NewTreeNodeValidator(failTreeNode),
		},
		{
			name:        "treeNodeValidator child",
			root:        ast.Root{Tree: &ast.TreeNode{Children: []*ast.TreeNode{{}}}},
			failMessage: "does not visit child TreeNodes",
			validator:   NewTreeNodeValidator(failLeafTreeNode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.root.Accept(tc.validator)

			if tc.validator.Error() == nil {
				t.Fatal(tc.failMessage)
			}
		})
	}
}
