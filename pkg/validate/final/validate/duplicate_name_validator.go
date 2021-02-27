package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DuplicateNames verifies that no objects share the same identifying tuple of:
// Group, Kind, metadata.namespace, metadata.name
func DuplicateNames(objs []ast.FileObject) status.MultiError {
	duplicateMap := make(map[groupKindNamespaceName][]ast.FileObject)
	for _, o := range objs {
		gknn := groupKindNamespaceName{
			group:     o.GroupVersionKind().Group,
			kind:      o.GroupVersionKind().Kind,
			namespace: o.GetNamespace(),
			name:      o.GetName(),
		}
		duplicateMap[gknn] = append(duplicateMap[gknn], o)
	}

	var errs status.MultiError
	for gknn, duplicates := range duplicateMap {
		if len(duplicates) > 1 {
			rs := resources(duplicates)
			if gknn.GroupKind() == kinds.Namespace().GroupKind() {
				errs = status.Append(errs, nonhierarchical.NamespaceCollisionError(gknn.name, rs...))
			} else if gknn.namespace == "" {
				errs = status.Append(errs, nonhierarchical.ClusterMetadataNameCollisionError(gknn.GroupKind(), gknn.name, rs...))
			} else {
				errs = status.Append(errs, nonhierarchical.NamespaceMetadataNameCollisionError(gknn.GroupKind(), gknn.namespace, gknn.name, rs...))
			}
		}
	}
	return errs
}

type groupKindNamespaceName struct {
	group     string
	kind      string
	namespace string
	name      string
}

// GroupKind is a convenience method to provide the GroupKind it contains.
func (gknn groupKindNamespaceName) GroupKind() schema.GroupKind {
	return schema.GroupKind{
		Group: gknn.group,
		Kind:  gknn.kind,
	}
}

func resources(objs []ast.FileObject) []id.Resource {
	rs := make([]id.Resource, len(objs))
	for i, obj := range objs {
		rs[i] = obj
	}
	return rs
}
