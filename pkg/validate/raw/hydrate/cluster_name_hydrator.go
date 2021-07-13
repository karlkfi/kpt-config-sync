package hydrate

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// ClusterName annotates the given Raw objects with the name of the current
// cluster.
func ClusterName(objs *objects.Raw) status.MultiError {
	if objs.ClusterName == "" {
		return nil
	}
	for _, obj := range objs.Objects {
		core.Annotation(metadata.ClusterNameAnnotationKey, objs.ClusterName)(obj)
	}
	return nil
}
