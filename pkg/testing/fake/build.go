package fake

import (
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Build constructs an ast.FileObject of the specified GroupVersionKind and applies the passed Mutators.
//
// If the type is one of the types supported in fake.go, returns an empty version of the specified
// object. Otherwise returns a FakeObject.
//
// Sets a default name for the object. The name may be removed by using the option Name("").
// Sets a default valid path for the object based on the kind. If the kind does not have a default path,
// it must be specified manually with Path("")
func Build(gvk schema.GroupVersionKind, opts ...object.Mutator) ast.FileObject {
	var o ast.FileObject
	switch gvk {
	case kinds.Cluster():
		o = Cluster("clusterregistry/cluster.yaml")
	case kinds.ClusterConfig():
		o = ClusterConfig()
	case kinds.ClusterRole():
		o = ClusterRole("cluster/cr.yaml")
	case kinds.ClusterSelector():
		o = ClusterSelector("cluster/cs.yaml")
	case kinds.Deployment():
		o = Deployment()
	case kinds.HierarchyConfig():
		o = HierarchyConfig("system/hc.yaml")
	case kinds.Namespace():
		o = Namespace("namespaces/foo/namespace.yaml")
	case kinds.NamespaceConfig():
		o = NamespaceConfig()
	case kinds.NamespaceSelector():
		o = NamespaceSelector("namespaces/foo/ns.yaml")
	case kinds.PersistentVolume():
		o = PersistentVolume()
	case kinds.Repo():
		o = Repo("system/repo.yaml")
	case kinds.Role():
		o = Role("namespaces/foo/role.yaml")
	case kinds.RoleBinding():
		o = RoleBinding("namespaces/foo/rb.yaml")
	default:
		o = asttesting.NewFakeFileObject(gvk, "")
	}

	// defaults are modifications which are made by default to all objects.
	var defaults = []object.Mutator{
		// Underlying implementations of meta.v1.Object inconsistently implement SetAnnotations and
		// SetLabels behavior on nil and when being initialized, so this guarantees tests will always
		// operate from the same state.
		object.Annotations(map[string]string{}),
		object.Labels(map[string]string{}),
		object.Name(strings.ToLower(gvk.Kind)).If(func(o ast.FileObject) bool {
			return o.Name() == ""
		})}

	opts = append(defaults, opts...)

	object.Mutate(opts...)(&o)
	return o
}
