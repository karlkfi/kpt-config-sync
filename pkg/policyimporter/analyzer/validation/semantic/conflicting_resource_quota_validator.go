package semantic

import (
	"path"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/validator"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/util/multierror"
)

var _ validator.Validator = ConflictingResourceQuotaValidator{}

// ConflictingResourceQuotaValidator ensures no more than one ResourceQuota is defined in a
// directory.  objects are *all* objects that need to be validated (across namespaces), and
// coverage is the coverage analyzer for per-cluster mapping.
type ConflictingResourceQuotaValidator struct {
	// objects contain all objects to validate for proper coverage
	objects []ast.FileObject
	// coverage is the examiner of object-to-cluster coverage
	coverage *coverage.ForCluster
}

// NewConflictingResourceQuotaValidator creates a new quota validator. objects
// are the objects to be validated.
func NewConflictingResourceQuotaValidator(
	objects []ast.FileObject, cov *coverage.ForCluster) ConflictingResourceQuotaValidator {
	return ConflictingResourceQuotaValidator{objects: objects, coverage: cov}
}

// Used as map key below
type dirCluster struct {
	dir, cluster string
}

// Validate adds errors to the errorBuilder if there are conflicting ResourceQuotas in a directory
func (v ConflictingResourceQuotaValidator) Validate(errorBuilder *multierror.Builder) {
	// Classify the ResourceQuota objects by the directory and cluster they are
	// targeted to.  Once done, make sure that each cluster gets only one such
	// object (1) per each unique path-cluster combination; (2) per each case
	// where there is a path-cluster combination that has a targeted RQ object
	// AND a RQ object that is targeted to all clusters for that same path.
	resourceQuotas := make(map[dirCluster][]id.Resource)
	for i, obj := range v.objects {
		if glog.V(5) {
			glog.V(5).Infof("obj: %v", obj)
		}
		if obj.GroupVersionKind() == kinds.ResourceQuota() {
			dir := path.Dir(obj.RelativeSlashPath())
			for _, c := range v.coverage.MapToClusters(obj.MetaObject()) {
				if glog.V(7) {
					glog.Infof("seen cluster: i=%v, dir=%v, c=%v", obj, dir, c)
				}
				dc := dirCluster{
					dir:     dir,
					cluster: c,
				}
				resourceQuotas[dc] = append(resourceQuotas[dc], &v.objects[i])
			}
		}
	}

	for dc, quotas := range resourceQuotas {
		if len(quotas) > 1 {
			errorBuilder.Add(veterrors.ConflictingResourceQuotaError{
				Path:       dc.dir,
				Cluster:    dc.cluster,
				Duplicates: quotas},
			)
		}
		// Also check for a conflict for the same path and "all" clusters.  This is a
		// special case when there's a 1 quota for this cluster and 1 quota targeted to
		// all clusters.  All other combinations (2 quotas here, 1 in all) are handled
		// by the default handler above.
		allQuotas := resourceQuotas[dirCluster{dir: dc.dir, cluster: ""}]
		if dc.cluster != "" && len(quotas) == 1 && len(allQuotas) == 1 {
			errorBuilder.Add(veterrors.ConflictingResourceQuotaError{
				Path:       dc.dir,
				Cluster:    dc.cluster,
				Duplicates: append(quotas, allQuotas...)},
			)
		}
	}
}
