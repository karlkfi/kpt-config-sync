package hydrate

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"github.com/google/nomos/pkg/webhook/configuration"
)

// DeclaredVersion annotates the given Raw objects with the API Version the
// object was declared in the repository.
func DeclaredVersion(objs *objects.Raw) status.MultiError {
	for _, obj := range objs.Objects {
		core.Label(configuration.DeclaredVersionLabel, obj.GroupVersionKind().Version)(obj)
	}
	return nil
}
