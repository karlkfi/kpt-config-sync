package object

import (
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BuildOpt modifies an ast.FileObject for testing.
type BuildOpt func(o *ast.FileObject)

// Build constructs an ast.FileObject of the specified type at the requested path.
// If the type is one of the types supported in fake.go, returns an empty version of the specified
// object.
// Sets a default name for the object. The name may be removed by using the option Name("").
// Sets a default valid path for the object based on the kind. If the kind does not have a default path,
// it must be specified manually with Path("")
func Build(gvk schema.GroupVersionKind, opts ...BuildOpt) ast.FileObject {
	var object ast.FileObject
	switch gvk {
	case kinds.Cluster():
		object = fake.Cluster("clusterregistry/cluster.yaml")
	case kinds.ClusterConfig():
		object = fake.ClusterConfig()
	case kinds.ClusterRole():
		object = fake.ClusterRole("cluster/cr.yaml")
	case kinds.ClusterSelector():
		object = fake.ClusterSelector("cluster/cs.yaml")
	case kinds.HierarchyConfig():
		object = fake.HierarchyConfig("system/hc.yaml")
	case kinds.Namespace():
		object = fake.Namespace("namespaces/foo/namespace.yaml")
	case kinds.NamespaceSelector():
		object = fake.NamespaceSelector("namespaces/foo/ns.yaml")
	case kinds.PersistentVolume():
		object = fake.PersistentVolume()
	case kinds.Repo():
		object = fake.Repo("system/repo.yaml")
	case kinds.Role():
		object = fake.Role("namespaces/foo/role.yaml")
	case kinds.RoleBinding():
		object = fake.RoleBinding("namespaces/foo/rb.yaml")
	default:
		object = asttesting.NewFakeFileObject(gvk, "")
	}

	if object.Name() == "" {
		Name(strings.ToLower(gvk.Kind))(&object)
	}
	// Underlying implementations of meta.v1.Object inconsistently implement SetAnnotations and
	// SetLabels behavior on nil and when being initialized, so this guarantees tests will always
	// operate from the same state.
	object.MetaObject().SetAnnotations(map[string]string{})
	object.MetaObject().SetLabels(map[string]string{})

	for _, opt := range opts {
		opt(&object)
	}

	return object
}

// Path replaces the path with the provided slash-delimited path from nomos root.
func Path(path string) BuildOpt {
	return func(o *ast.FileObject) {
		o.Path = nomospath.FromSlash(path)
	}
}
