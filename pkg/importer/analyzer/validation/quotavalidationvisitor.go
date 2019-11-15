package validation

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	corev1 "k8s.io/api/core/v1"
)

// QuotaValidator checks that ResourceQuota doesn't set scope related fields.
type QuotaValidator struct {
	*visitor.Base
	errs status.MultiError
}

var _ ast.Visitor = &QuotaValidator{}

// NewQuotaValidator creates a new validator.
func NewQuotaValidator() *QuotaValidator {
	v := &QuotaValidator{
		Base: visitor.NewBase(),
	}
	v.Base.SetImpl(v)

	return v
}

// Error returns any errors encountered during processing
func (v *QuotaValidator) Error() status.MultiError {
	return v.errs
}

// VisitObject implements Visitor
func (v *QuotaValidator) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	if o.GroupVersionKind() == kinds.ResourceQuota() {
		quota := *o.FileObject.Object.(*corev1.ResourceQuota)
		// Scope-related fields aren't supported by the merge so error pre-emptively if set.
		if quota.Spec.Scopes != nil {
			v.errs = status.Append(v.errs, IllegalResourceQuotaFieldError(
				o,
				"scopes"))
		}
		if quota.Spec.ScopeSelector != nil {
			v.errs = status.Append(v.errs, IllegalResourceQuotaFieldError(
				o,
				"scopeSelector"))
		}
	}

	return v.Base.VisitObject(o)
}

// IllegalResourceQuotaFieldErrorCode is the error code for llegalResourceQuotaFieldError
const IllegalResourceQuotaFieldErrorCode = "1008"

var illegalResourceQuotaFieldError = status.NewErrorBuilder(IllegalResourceQuotaFieldErrorCode)

// IllegalResourceQuotaFieldError represents illegal fields set on ResourceQuota objects.
func IllegalResourceQuotaFieldError(resource id.Resource, field string) status.Error {
	return illegalResourceQuotaFieldError.Sprintf(
		"A ResourceQuota config MUST NOT set scope when hierarchyMode is set to hierarchicalQuota. "+
			"Remove illegal field %s from:",
		field).BuildWithResources(resource)
}
