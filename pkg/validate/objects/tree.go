package objects

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TreeVisitor is a function that validates or hydrates Raw objects.
type TreeVisitor func(t *Tree) status.MultiError

// Tree contains a collection of FileObjects that are organized based upon the
// structure of a hierarchical repo. This includes system-level objects like
// HierarchyConfigs as well as a hierarchical tree of namespaces and namespace-
// scoped objects.
type Tree struct {
	Cluster          []ast.FileObject
	Tree             *ast.TreeNode
	Repo             ast.FileObject
	HierarchyConfigs []ast.FileObject
}

// BuildTree builds a Tree collection of objects from the given Scoped objects.
func BuildTree(scoped *Scoped) (*Tree, status.MultiError) {
	var errs status.MultiError
	t := &Tree{}

	// First process cluster-scoped resources.
	for _, obj := range scoped.Cluster {
		dir, err := topLevelDirectory(obj, repo.ClusterDir)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}

		if dir == repo.ClusterDir {
			t.Cluster = append(t.Cluster, obj)
		} else if err = t.addSystemObj(obj); err != nil {
			errs = status.Append(errs, err)
		}
	}

	// Next, do our best to process unknown-scoped resources.
	var namespaceObjs []ast.FileObject
	for _, obj := range scoped.Unknown {
		sourcePath := obj.Relative.OSPath()
		dir := cmpath.RelativeSlash(sourcePath).Split()[0]
		switch dir {
		case repo.SystemDir:
			if err := t.addSystemObj(obj); err != nil {
				errs = status.Append(errs, err)
			}
		case repo.ClusterDir:
			t.Cluster = append(t.Cluster, obj)
		case repo.NamespacesDir:
			namespaceObjs = append(namespaceObjs, obj)
		default:
			errs = status.Append(errs, status.InternalErrorf("unhandled top level directory: %q", dir))
		}
	}

	// Finally, process namespace-scoped resources.
	for _, obj := range scoped.Namespace {
		_, err := topLevelDirectory(obj, repo.NamespacesDir)
		if err != nil {
			errs = status.Append(errs, err)
		}
	}
	v := tree.NewBuilderVisitor(append(namespaceObjs, scoped.Namespace...))
	astRoot := v.VisitRoot(&ast.Root{})
	t.Tree = astRoot.Tree
	errs = status.Append(errs, v.Error())

	if errs != nil {
		return nil, errs
	}
	return t, nil
}

func (t *Tree) addSystemObj(obj ast.FileObject) status.Error {
	gk := obj.GroupVersionKind().GroupKind()
	switch gk {
	case kinds.HierarchyConfig().GroupKind():
		t.HierarchyConfigs = append(t.HierarchyConfigs, obj)
	case kinds.Repo().GroupKind():
		// We have already validated that there is only one Repo object.
		t.Repo = obj
	default:
		return status.InternalErrorf("unhandled system object: %v", obj)
	}
	return nil
}

var topLevelDirectoryOverrides = map[schema.GroupVersionKind]string{
	kinds.Repo():            repo.SystemDir,
	kinds.HierarchyConfig(): repo.SystemDir,

	kinds.Namespace():         repo.NamespacesDir,
	kinds.NamespaceSelector(): repo.NamespacesDir,
}

func topLevelDirectory(obj ast.FileObject, expectedDir string) (string, status.Error) {
	gvk := obj.GroupVersionKind()
	if override, hasOverride := topLevelDirectoryOverrides[gvk]; hasOverride {
		expectedDir = override
	}

	sourcePath := obj.Relative.OSPath()
	if cmpath.RelativeSlash(sourcePath).Split()[0] == expectedDir {
		return expectedDir, nil
	}

	switch expectedDir {
	case repo.SystemDir:
		return "", validation.ShouldBeInSystemError(obj)
	case repo.ClusterDir:
		return "", validation.ShouldBeInClusterError(obj)
	case repo.NamespacesDir:
		return "", validation.ShouldBeInNamespacesError(obj)
	default:
		return "", status.InternalErrorf("unhandled top level directory: %q", expectedDir)
	}
}
