package visitor

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

func testError() error {
	return vet.UndocumentedError("error")
}

func fakeObject() ast.FileObject {
	return asttesting.NewFakeFileObject(kinds.Role(), "")
}

func failRoot(_ *ast.Root) error {
	return testError()
}

func failSystem(_ *ast.System) error {
	return testError()
}

func failSystemObject(_ *ast.SystemObject) error {
	return testError()
}

func failClusterRegistry(_ *ast.ClusterRegistry) error {
	return testError()
}

func failClusterRegistryObject(_ *ast.ClusterRegistryObject) error {
	return testError()
}

func failCluster(_ *ast.Cluster) error {
	return testError()
}

func failClusterObject(_ *ast.ClusterObject) error {
	return testError()
}

func failTreeNode(_ *ast.TreeNode) error {
	return testError()
}

func failLeafTreeNode(n *ast.TreeNode) error {
	if len(n.Children) == 0 {
		return testError()
	}
	return nil
}

func failObject(_ *ast.NamespaceObject) error {
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
			root:        ast.Root{System: &ast.System{}},
			failMessage: "does not visit System",
			validator:   NewSystemValidator(failSystem),
		},
		{
			name:        "systemObjectValidator",
			root:        ast.Root{System: &ast.System{Objects: []*ast.SystemObject{{FileObject: fakeObject()}}}},
			failMessage: "does not visit System objects",
			validator:   NewSystemObjectValidator(failSystemObject),
		},
		{
			name:        "clusterRegistryValidator",
			root:        ast.Root{ClusterRegistry: &ast.ClusterRegistry{}},
			failMessage: "does not visit ClusterRegistry",
			validator:   NewClusterRegistryValidator(failClusterRegistry),
		},
		{
			name:        "clusterRegistryObjectValidator",
			root:        ast.Root{ClusterRegistry: &ast.ClusterRegistry{Objects: []*ast.ClusterRegistryObject{{FileObject: fakeObject()}}}},
			failMessage: "does not visit ClusterRegistry objects",
			validator:   NewClusterRegistryObjectValidator(failClusterRegistryObject),
		},
		{
			name:        "clusterValidator",
			root:        ast.Root{Cluster: &ast.Cluster{}},
			failMessage: "does not visit Cluster",
			validator:   NewClusterValidator(failCluster),
		},
		{
			name:        "clusterObjectValidator",
			root:        ast.Root{Cluster: &ast.Cluster{Objects: []*ast.ClusterObject{{FileObject: fakeObject()}}}},
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
