package semantic

import (
	"path"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
)

// ConflictingResourceQuotaValidator ensures no more than one ResourceQuota is defined in a
// directory.
type ConflictingResourceQuotaValidator struct {
	Objects []ast.FileObject
}

// Validate adds errors to the errorBuilder if there are conflicting ResourceQuotas in a directory
func (v ConflictingResourceQuotaValidator) Validate(errorBuilder *multierror.Builder) {
	resourceQuotas := make(map[string][]ast.FileObject)

	for _, obj := range v.Objects {
		if obj.GroupVersionKind() == corev1.SchemeGroupVersion.WithKind("ResourceQuota") {
			dir := path.Dir(obj.Source)
			resourceQuotas[dir] = append(resourceQuotas[dir], obj)
		}
	}

	for dir, quotas := range resourceQuotas {
		if len(quotas) > 1 {
			errorBuilder.Add(vet.ConflictingResourceQuotaError{Path: dir, Duplicates: quotas})
		}
	}
}
