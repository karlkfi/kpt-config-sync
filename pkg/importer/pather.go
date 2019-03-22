package importer

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Pather assigns default unique relative paths to objects with no path yet on a disk.
type Pather struct {
	namespaced map[schema.GroupVersionKind]bool
}

// NewPather creates a pather from the list of APIResources on the server.
func NewPather(resources ...metav1.APIResource) Pather {
	result := Pather{
		namespaced: make(map[schema.GroupVersionKind]bool),
	}
	for _, resource := range resources {
		gvk := schema.GroupVersionKind{
			Group:   resource.Group,
			Version: resource.Version,
			Kind:    resource.Kind,
		}
		result.namespaced[gvk] = resource.Namespaced
	}
	return result
}

const (
	repoBasePath      = "repo.yaml"
	namespaceBasePath = "namespace.yaml"
)

// systemKinds is a map from kinds to whether they are in system/. Note kinds not present below are
// implicitly false.
var systemKinds = map[schema.GroupVersionKind]bool{
	kinds.Repo():            true,
	kinds.HierarchyConfig(): true,
}

// clusterRegistryKinds is a map from kinds to whether they are in clusterregistry/. Note kinds not
// present below are implicitly false.
var clusterRegistryKinds = map[schema.GroupVersionKind]bool{
	kinds.Cluster():         true,
	kinds.ClusterSelector(): true,
}

// shortBase returns the short name for an object.
func shortBase(o ast.FileObject) string {
	gk := o.GroupVersionKind().GroupKind()

	switch gk {
	case kinds.Repo().GroupKind():
		// This means if there are multiple Repo objects in the cluster, we will just pick one.
		// The one we pick is undefined.
		return repoBasePath
	case kinds.Namespace().GroupKind():
		return namespaceBasePath
	}

	kind := strings.ToLower(o.GroupVersionKind().Kind)
	name := o.Name()

	return fmt.Sprintf("%s_%s.yaml", kind, name)
}

// longBase returns the long name for an object to use if there are file name collisions.
func longBase(o ast.FileObject) string {
	kind := strings.ToLower(o.GroupVersionKind().Kind)
	group := o.GroupVersionKind().Group
	name := o.Name()

	return fmt.Sprintf("%s_%s_%s.yaml", kind, group, name)
}

// directory returns the relative directory to write the object to.
func (p Pather) directory(o ast.FileObject) cmpath.Path {
	gvk := o.GroupVersionKind()
	switch {
	case systemKinds[gvk]:
		return cmpath.FromSlash(repo.SystemDir)
	case clusterRegistryKinds[gvk]:
		return cmpath.FromSlash(repo.ClusterRegistryDir)
	case gvk == kinds.Namespace():
		return cmpath.FromSlash(repo.NamespacesDir).Join(o.MetaObject().GetName())
	case p.namespaced[gvk]:
		return cmpath.FromSlash(repo.NamespacesDir).Join(o.MetaObject().GetNamespace())
	default:
		return cmpath.FromSlash(repo.ClusterDir)
	}
}

// AddPaths adds the expected relative paths to write the files to. Paths are guaranteed to be
// unique for a valid collection of objects. Behavior undefined for collections of objects which
// could not validly be in a single cluster.
func (p Pather) AddPaths(objects []ast.FileObject) {
	shortPathCounts := make(map[cmpath.Path]int)

	for i, object := range objects {
		if object.Object == nil {
			continue
		}
		objectPath := p.directory(object).Join(shortBase(object))
		objects[i].Path = objectPath
		shortPathCounts[objectPath]++
	}

	for i, object := range objects {
		if shortPathCounts[object.Path] > 1 {
			objects[i].Path = p.directory(object).Join(longBase(object))
		}
	}
}
