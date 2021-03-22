package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceSelectors validates that all NamespaceSelectors have a unique name.
func NamespaceSelectors(objs *objects.Scoped) status.MultiError {
	var errs status.MultiError
	gk := kinds.NamespaceSelector().GroupKind()
	matches := make(map[string][]client.Object)

	for _, obj := range objs.Cluster {
		if obj.GetObjectKind().GroupVersionKind().GroupKind() == gk {
			matches[obj.GetName()] = append(matches[obj.GetName()], obj)
		}
	}

	for name, duplicates := range matches {
		if len(duplicates) > 1 {
			errs = status.Append(errs, nonhierarchical.SelectorMetadataNameCollisionError(gk.Kind, name, duplicates...))
		}
	}

	return errs
}
