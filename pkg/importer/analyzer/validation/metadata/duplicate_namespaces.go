package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// listNamespaces returns a list of id.Resources representing the Namespaces the TreeNode contains.
func listNamespaces(r *ast.TreeNode) []id.Resource {
	// Handles case when the root node is nil.
	if r == nil {
		// Trivially, there are no Namespaces in an empty TreeNode.
		return nil
	}

	var result []id.Resource
	if r.Type == node.Namespace {
		result = append(result, namespaceNode{r})
	}
	for _, child := range r.Children {
		result = append(result, listNamespaces(child)...)
	}
	return result
}

// namespaceNode wraps TreeNode to make it meet the id.Resource interface.
// The Namespace TreeNode has all of the metadata from the Namespace we discarded earlier in parsing.
type namespaceNode struct {
	*ast.TreeNode
}

var _ id.Resource = namespaceNode{}

// GetNamespace implements id.Resource.
func (n namespaceNode) GetNamespace() string {
	// Namespaces are cluster-scoped, so their metadata.namespace is undefined.
	return ""
}

// GetName implements id.Resource.
func (n namespaceNode) GetName() string {
	return n.TreeNode.Name()
}

// GroupVersionKind implements id.Resource.
func (n namespaceNode) GroupVersionKind() schema.GroupVersionKind {
	return kinds.Namespace()
}
