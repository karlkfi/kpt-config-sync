package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
)

// DisallowedFields verifies if the given Raw objects contain any fields which
// are not allowed to be declared in Git.
func DisallowedFields(objs *objects.Raw) status.MultiError {
	var errs status.MultiError
	for _, obj := range objs.Objects {
		if len(obj.GetOwnerReferences()) > 0 {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.OwnerReference))
		}
		if obj.GetSelfLink() != "" {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.SelfLink))
		}
		if obj.GetUID() != "" {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.UID))
		}
		if obj.GetResourceVersion() != "" {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.ResourceVersion))
		}
		if obj.GetGeneration() != 0 {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.Generation))
		}
		if !obj.GetCreationTimestamp().Time.IsZero() {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.CreationTimestamp))
		}
		if obj.GetDeletionTimestamp() != nil {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.DeletionTimestamp))
		}
		if obj.GetDeletionGracePeriodSeconds() != nil {
			errs = status.Append(errs, syntax.IllegalFieldsInConfigError(obj, id.DeletionGracePeriodSeconds))
		}
	}
	return errs
}
