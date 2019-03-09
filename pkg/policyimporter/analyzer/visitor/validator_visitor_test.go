package visitor

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/status"
)

func testError() *status.MultiError {
	return status.From(status.UndocumentedError("error"))
}

func fakeObject() ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Role(), "")
}

func failRoot(_ *ast.Root) *status.MultiError {
	return testError()
}

func failSystem(_ *ast.System) *status.MultiError {
	return testError()
}

func failSystemObject(_ *ast.SystemObject) *status.MultiError {
	return testError()
}

func failClusterRegistry(_ *ast.ClusterRegistry) *status.MultiError {
	return testError()
}

func failClusterRegistryObject(_ *ast.ClusterRegistryObject) *status.MultiError {
	return testError()
}

func failCluster(_ *ast.Cluster) *status.MultiError {
	return testError()
}

func failClusterObject(_ *ast.ClusterObject) *status.MultiError {
	return testError()
}

func failTreeNode(_ *ast.TreeNode) *status.MultiError {
	return testError()
}

func failLeafTreeNode(n *ast.TreeNode) *status.MultiError {
	if len(n.Children) == 0 {
		return testError()
	}
	return nil
}

func failObject(_ *ast.NamespaceObject) *status.MultiError {
	return testError()
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
			name:        "systemValidator",
			root:        ast.Root{},
			failMessage: "does not visit System",
			validator:   NewSystemValidator(failSystem),
		},
		{
			name:        "systemObjectValidator",
			root:        ast.Root{SystemObjects: []*ast.SystemObject{{FileObject: fakeObject()}}},
			failMessage: "does not visit System objects",
			validator:   NewSystemObjectValidator(failSystemObject),
		},
		{
			name:        "clusterRegistryValidator",
			root:        ast.Root{},
			failMessage: "does not visit ClusterRegistry",
			validator:   NewClusterRegistryValidator(failClusterRegistry),
		},
		{
			name:        "clusterRegistryObjectValidator",
			root:        ast.Root{ClusterRegistryObjects: []*ast.ClusterRegistryObject{{FileObject: fakeObject()}}},
			failMessage: "does not visit ClusterRegistry objects",
			validator:   NewClusterRegistryObjectValidator(failClusterRegistryObject),
		},
		{
			name:        "clusterValidator",
			root:        ast.Root{},
			failMessage: "does not visit Cluster",
			validator:   NewClusterValidator(failCluster),
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
		{
			name:        "objectValidator",
			root:        ast.Root{Tree: &ast.TreeNode{Objects: []*ast.NamespaceObject{{FileObject: fakeObject()}}}},
			failMessage: "does not visit root TreeNode objects",
			validator:   NewObjectValidator(failObject),
		},
		{
			name:        "objectValidator",
			root:        ast.Root{Tree: &ast.TreeNode{Children: []*ast.TreeNode{{Objects: []*ast.NamespaceObject{{FileObject: fakeObject()}}}}}},
			failMessage: "does not visit child TreeNode objects",
			validator:   NewObjectValidator(failObject),
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
