package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// NewDisallowedFieldsValidator validates that imported objects do not contain disallowed fields.
func NewDisallowedFieldsValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		m := o.MetaObject()
		if len(m.GetOwnerReferences()) > 0 {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.OwnerReference))
		}
		if m.GetSelfLink() != "" {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.SelfLink))
		}
		if m.GetUID() != "" {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.UID))
		}
		if m.GetResourceVersion() != "" {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.ResourceVersion))
		}
		if m.GetGeneration() != 0 {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.Generation))
		}
		if !m.GetCreationTimestamp().Time.IsZero() {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.CreationTimestamp))
		}
		if m.GetDeletionTimestamp() != nil {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.DeletionTimestamp))
		}
		if m.GetDeletionGracePeriodSeconds() != nil {
			return status.From(vet.IllegalFieldsInConfigError(&o, id.DeletionGracePeriodSeconds))
		}

		return nil
	})
}
