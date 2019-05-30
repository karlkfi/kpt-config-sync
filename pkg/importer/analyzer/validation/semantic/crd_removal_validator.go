package semantic

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// CRDRemovalValidator validates that a CRD is not being removed from the repo,
// while its corresponding Custom Resources still exist on the cluster.
type CRDRemovalValidator struct {
	*visitor.ValidatorBase
	crdInfo *clusterconfig.CRDInfo
}

// NewCRDRemovalValidator instantiates an CRDRemovalValidator.
func NewCRDRemovalValidator() ast.Visitor {
	return visitor.NewValidator(&CRDRemovalValidator{})
}

// ValidateRoot adds CRDClusterConfigInfo to Root Extensions.
func (v *CRDRemovalValidator) ValidateRoot(r *ast.Root) status.MultiError {
	crdInfo, err := clusterconfig.GetCRDInfo(r)
	v.crdInfo = crdInfo
	return status.From(err)
}

// ValidateClusterObject implements Visitor.
func (v *CRDRemovalValidator) ValidateClusterObject(o *ast.ClusterObject) status.MultiError {
	return v.validate(o.FileObject)
}

// ValidateObject implements Visitor.
func (v *CRDRemovalValidator) ValidateObject(o *ast.NamespaceObject) status.MultiError {
	return v.validate(o.FileObject)
}

func (v *CRDRemovalValidator) validate(o ast.FileObject) status.MultiError {
	if crd, pendingRemoval := v.crdInfo.CRDPendingRemoval(o); pendingRemoval {
		return status.From(vet.UnsupportedCRDRemovalError(ast.ParseFileObject(crd)))
	}
	return nil
}
