package treetesting

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree"
)

// BuildTree creates and populates an ast.Root with the provided objects.
// Assumes all objects are in the same top-level directory, and that top-level directory is the
// hierarchical policy directory.
func BuildTree(objects ...ast.FileObject) *ast.Root {
	// TODO: add logic specific to:
	//  1. cluster/
	//  2. clusterregistry/
	//  3. namespaces/ + hierarchy/
	//  4. system/
	root := &ast.Root{}
	root.Accept(tree.NewBuilderVisitor(objects))

	return root
}
