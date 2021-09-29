package hydrate

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// UnknownScope hydrates the given Scoped objects by adding an annotation
// `configsync.gke.io/unknown-scope: true` into every object whose scope is
// unknown.
func UnknownScope(objs *objects.Scoped) status.MultiError {
	for _, obj := range objs.Unknown {
		core.SetAnnotation(obj, metadata.UnknownScopeAnnotationKey, metadata.UnknownScopeAnnotationValue)
	}
	return nil
}
