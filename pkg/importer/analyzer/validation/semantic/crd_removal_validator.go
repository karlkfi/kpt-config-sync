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
	return err
}

// ValidateClusterObject implements Visitor.
func (v *CRDRemovalValidator) ValidateClusterObject(o *ast.ClusterObject) status.MultiError {
	return CheckCRDPendingRemoval(v.crdInfo, o.FileObject)
}

// ValidateObject implements Visitor.
func (v *CRDRemovalValidator) ValidateObject(o *ast.NamespaceObject) status.MultiError {
	return CheckCRDPendingRemoval(v.crdInfo, o.FileObject)
}

// CheckCRDPendingRemoval returns an error if the type is from a CRD pending removal.
func CheckCRDPendingRemoval(crdInfo *clusterconfig.CRDInfo, o ast.FileObject) status.Error {
	if crd, pendingRemoval := crdInfo.CRDPendingRemoval(o); pendingRemoval {
		return vet.UnsupportedCRDRemovalError(ast.ParseFileObject(crd))
	}
	return nil
}
