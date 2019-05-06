package tree

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

// CRDClusterConfigInfoVisitor adds ClusterConfigInfo to Extensions.
type CRDClusterConfigInfoVisitor struct {
	*visitor.Base
	crdInfo *clusterconfig.CRDInfo
	errs    status.MultiError
}

// NewCRDClusterConfigInfoVisitor instantiates an CRDClusterConfigInfoVisitor with a set of objects to add.
func NewCRDClusterConfigInfoVisitor(crdInfo *clusterconfig.CRDInfo) *CRDClusterConfigInfoVisitor {
	v := &CRDClusterConfigInfoVisitor{
		Base:    visitor.NewBase(),
		crdInfo: crdInfo,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds CRDInfo to Root Extensions.
func (v *CRDClusterConfigInfoVisitor) VisitRoot(r *ast.Root) *ast.Root {
	v.errs = status.Append(v.errs, clusterconfig.AddCRDInfo(r, v.crdInfo))
	return r
}

// Error implements Visitor.
func (v *CRDClusterConfigInfoVisitor) Error() status.MultiError {
	return v.errs
}
