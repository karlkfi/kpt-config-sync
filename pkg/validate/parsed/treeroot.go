package parsed

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// TreeRoot is a structured, hierarchical collection of declared configs.
type TreeRoot struct {
	// ClusterObjects represents resources that are cluster-scoped.
	ClusterObjects []ast.FileObject
	// ClusterRegistryObjects represents resources that are related to multi-cluster.
	ClusterRegistryObjects []ast.FileObject
	// SystemObjects represents resources regarding ConfigSync configuration.
	SystemObjects []ast.FileObject
	// Tree represents the directory hierarchy containing namespace scoped resources.
	Tree *ast.TreeNode
}

var _ Root = &TreeRoot{}

// VisitAllObjects implements Root.
func (t *TreeRoot) VisitAllObjects(visit VisitorFunc) status.MultiError {
	err := t.VisitSystemObjects(visit)
	err = status.Append(err, t.VisitClusterRegistryObjects(visit))
	err = status.Append(err, t.VisitClusterObjects(visit))
	return status.Append(err, t.VisitNamespaceObjects(visit))
}

// VisitClusterObjects implements Root.
func (t *TreeRoot) VisitClusterObjects(visit VisitorFunc) status.MultiError {
	return visit(t.ClusterObjects)
}

// VisitClusterRegistryObjects implements Root.
func (t *TreeRoot) VisitClusterRegistryObjects(visit VisitorFunc) status.MultiError {
	return visit(t.ClusterRegistryObjects)
}

// VisitNamespaceObjects implements Root.
func (t *TreeRoot) VisitNamespaceObjects(visit VisitorFunc) status.MultiError {
	if t.Tree == nil {
		return nil
	}
	return visitTreeNode(visit, t.Tree)
}

// VisitSystemObjects implements Root.
func (t *TreeRoot) VisitSystemObjects(visit VisitorFunc) status.MultiError {
	return visit(t.SystemObjects)
}

func visitTreeNode(visit VisitorFunc, node *ast.TreeNode) status.MultiError {
	var objs []ast.FileObject
	for _, o := range node.Objects {
		objs = append(objs, o.FileObject)
	}
	err := visit(objs)

	for _, c := range node.Children {
		err = status.Append(err, visitTreeNode(visit, c))
	}
	return err
}

// BuildTree builds a new TreeRoot from the given Root (typically a FlatRoot).
func BuildTree(from Root) (*TreeRoot, status.MultiError) {
	root := &TreeRoot{}
	err := status.Append(
		from.VisitSystemObjects(systemVisitor(root)),
		from.VisitClusterRegistryObjects(clusterRegistryVisitor(root)),
		from.VisitClusterObjects(clusterVisitor(root)))
	// Building the Tree requires valid state, so we check for errors first and exit early.
	if err != nil {
		return nil, err
	}
	if err = from.VisitNamespaceObjects(namespaceVisitor(root)); err != nil {
		return nil, err
	}
	return root, nil
}

func systemVisitor(root *TreeRoot) VisitorFunc {
	return func(objs []ast.FileObject) status.MultiError {
		foundRepo := false
		for _, o := range objs {
			if o.GroupVersionKind() == kinds.Repo() {
				foundRepo = true
			}
			root.SystemObjects = append(root.SystemObjects, o)
		}
		if !foundRepo {
			return system.MissingRepoError()
		}
		return nil
	}
}

func clusterRegistryVisitor(root *TreeRoot) VisitorFunc {
	return PerObjectVisitor(func(obj ast.FileObject) status.Error {
		root.ClusterRegistryObjects = append(root.ClusterRegistryObjects, obj)
		return nil
	})
}

func clusterVisitor(root *TreeRoot) VisitorFunc {
	return PerObjectVisitor(func(obj ast.FileObject) status.Error {
		root.ClusterObjects = append(root.ClusterObjects, obj)
		return nil
	})
}

func namespaceVisitor(root *TreeRoot) VisitorFunc {
	return func(objs []ast.FileObject) status.MultiError {
		v := tree.NewBuilderVisitor(objs)
		astRoot := v.VisitRoot(&ast.Root{})
		root.Tree = astRoot.Tree
		return v.Error()
	}
}
