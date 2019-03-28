package syntax

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDisallowedFieldsValidator validates that imported objects do not contain disallowed fields.
func NewDisallowedFieldsValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) *status.MultiError {
		m := o.Object.(metav1.Object)
		if len(m.GetOwnerReferences()) > 0 {
			return status.From(vet.IllegalFieldsInConfigError{Resource: &o, Field: id.OwnerReference})
		}
		return nil
	})
}
