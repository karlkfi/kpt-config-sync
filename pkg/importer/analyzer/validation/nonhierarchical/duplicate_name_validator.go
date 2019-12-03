package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DuplicateNameValidator forbids declaring conlficting resources.
var DuplicateNameValidator = validator{
	validate: func(objects []ast.FileObject) status.MultiError {
		resources := make([]id.Resource, len(objects))
		for i, o := range objects {
			obj := o // Use intermediate object since taking the reference of a loop variable is bad.
			resources[i] = &obj
		}
		return checkDuplicates(resources)
	},
}

// NameCollisionErrorCode is the error code for ObjectNameCollisionError
const NameCollisionErrorCode = "1029"

// nameCollisionErrorBuilder is
var nameCollisionErrorBuilder = status.NewErrorBuilder(NameCollisionErrorCode)

// NamespaceCollisionError reports multiple declared Namespaces with the same name.
func NamespaceCollisionError(name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.
		Sprintf("Namespaces MUST have unique names. Found %d Namespaces named %q. Rename or merge the Namespaces to fix:",
			len(duplicates), name).
		BuildWithResources(duplicates...)
}

// NamespaceMetadataNameCollisionError reports that multiple namespace-scoped objects of the same Kind and
// namespace have the same metadata name
func NamespaceMetadataNameCollisionError(gk schema.GroupKind, namespace string, name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.
		Sprintf("Namespace-scoped configs of the same Group and Kind MUST have unique names if they are in the same Namespace. "+
			"Found %d configs of GroupKind %q in Namespace %q named %q. Rename or delete the duplicates to fix:",
			len(duplicates), gk.String(), namespace, name).
		BuildWithResources(duplicates...)
}

// ClusterMetadataNameCollisionError reports that multiple cluster-scoped objects of the same Kind and
// namespace have the same metadata.name.
func ClusterMetadataNameCollisionError(gk schema.GroupKind, name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.
		Sprintf("Cluster-scoped configs of the same Group and Kind MUST have unique names. "+
			"Found %d configs of GroupKind %q named %q. Rename or delete the duplicates to fix:",
			len(duplicates), gk.String(), name).
		BuildWithResources(duplicates...)
}

// SelectorMetadataNameCollisionError reports that multiple ClusterSelectors or NamespaceSelectors
// have the same metadata.name.
func SelectorMetadataNameCollisionError(kind string, name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.
		Sprintf("%ss MUST have globally-unique names. "+
			"Found %d %ss named %q. Rename or delete the duplicates to fix:",
			kind, len(duplicates), kind, name).
		BuildWithResources(duplicates...)
}

// checkDuplicates returns an error if it detects multiple objects with the same Group, Kind,
// metadata.namespace, and metadata.name.
func checkDuplicates(objects []id.Resource) status.MultiError {
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
