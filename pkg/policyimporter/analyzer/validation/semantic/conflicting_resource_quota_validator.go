package semantic

import (
	"path"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/util/multierror"
)

var _ validator.Validator = ConflictingResourceQuotaValidator{}

// ConflictingResourceQuotaValidator ensures no more than one ResourceQuota is defined in a
// directory.  objects are *all* objects that need to be validated (across namespaces), and
// coverage is the coverage analyzer for per-cluster mapping.
type ConflictingResourceQuotaValidator struct {
	objects []ast.FileObject
}

// NewConflictingResourceQuotaValidator creates a new quota validator. objects
// are the objects to be validated.
func NewConflictingResourceQuotaValidator(objects []ast.FileObject) ConflictingResourceQuotaValidator {
	return ConflictingResourceQuotaValidator{objects: objects}
}

// Validate adds errors to the errorBuilder if there are conflicting ResourceQuotas in a directory
func (v ConflictingResourceQuotaValidator) Validate(errorBuilder *multierror.Builder) {
	resourceQuotas := make(map[string][]veterrors.ResourceID)

	for _, obj := range v.objects {
		if obj.GroupVersionKind() == kinds.ResourceQuota() {
			dir := path.Dir(obj.Source())
			resourceQuotas[dir] = append(resourceQuotas[dir], &obj)
		}
	}

	for dir, quotas := range resourceQuotas {
		if len(quotas) > 1 {
			errorBuilder.Add(veterrors.ConflictingResourceQuotaError{Path: dir, Duplicates: quotas})
		}
	}
}
