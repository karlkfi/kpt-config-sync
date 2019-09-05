package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

// IllegalFieldsInConfigErrorCode is the error code for IllegalFieldsInConfigError
const IllegalFieldsInConfigErrorCode = "1045"

func init() {
	replicaSet := fake.ReplicaSet()
	status.AddExamples(IllegalFieldsInConfigErrorCode,
		IllegalFieldsInConfigError(&replicaSet, id.OwnerReference))
}

// NewDisallowedFieldsValidator validates that imported objects do not contain disallowed fields.
func NewDisallowedFieldsValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		return DisallowFields(o)
	})
}

// DisallowFields returns an error if o contains any disallowed fields.
func DisallowFields(o ast.FileObject) status.Error {
	m := o.MetaObject()
	if len(m.GetOwnerReferences()) > 0 {
		return IllegalFieldsInConfigError(&o, id.OwnerReference)
	}
	if m.GetSelfLink() != "" {
		return IllegalFieldsInConfigError(&o, id.SelfLink)
	}
	if m.GetUID() != "" {
		return IllegalFieldsInConfigError(&o, id.UID)
	}
	if m.GetResourceVersion() != "" {
		return IllegalFieldsInConfigError(&o, id.ResourceVersion)
	}
	if m.GetGeneration() != 0 {
		return IllegalFieldsInConfigError(&o, id.Generation)
	}
	if !m.GetCreationTimestamp().Time.IsZero() {
		return IllegalFieldsInConfigError(&o, id.CreationTimestamp)
	}
	if m.GetDeletionTimestamp() != nil {
		return IllegalFieldsInConfigError(&o, id.DeletionTimestamp)
	}
	if m.GetDeletionGracePeriodSeconds() != nil {
		return IllegalFieldsInConfigError(&o, id.DeletionGracePeriodSeconds)
	}
	return nil
}

var illegalFieldsInConfigErrorBuilder = status.NewErrorBuilder(IllegalFieldsInConfigErrorCode)

// IllegalFieldsInConfigError reports that an object has an illegal field set.
func IllegalFieldsInConfigError(resource id.Resource, field id.DisallowedField) status.Error {
	return illegalFieldsInConfigErrorBuilder.WithResources(resource).Errorf(
		"Configs with %[1]q specified are not allowed. "+
			"To fix, either remove the config or remove the %[1]q field in the config:",
		field)
}
