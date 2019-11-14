package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CheckDuplicates returns an error if it detects multiple objects with the same Group, Kind,
// metadata.namespace, and metadata.name.
func CheckDuplicates(objects []id.Resource) status.MultiError {
	duplicateMap := make(map[groupKindNamespaceName][]id.Resource)

	for _, o := range objects {
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
			if gknn.GroupKind() == kinds.Namespace().GroupKind() {
				errs = status.Append(errs, NamespaceCollisionError(gknn.name, duplicates...))
			} else if gknn.namespace == "" {
				errs = status.Append(errs, ClusterMetadataNameCollisionError(gknn.GroupKind(), gknn.name, duplicates...))
			} else {
				errs = status.Append(errs, NamespaceMetadataNameCollisionError(gknn.GroupKind(), gknn.namespace, gknn.name, duplicates...))
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

type duplicateNameValidator struct {
	visitor.ValidatorBase
}
