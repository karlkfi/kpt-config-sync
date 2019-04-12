package tree

import (
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

// CRDClusterConfigInfoVisitor adds ClusterConfigInfo to Extensions.
type CRDClusterConfigInfoVisitor struct {
	*visitor.Base
	crdInfo *importer.CRDClusterConfigInfo
	errs    status.MultiError
}

// NewCRDClusterConfigInfoVisitor instantiates an CRDClusterConfigInfoVisitor with a set of objects to add.
func NewCRDClusterConfigInfoVisitor(crdInfo *importer.CRDClusterConfigInfo) *CRDClusterConfigInfoVisitor {
	v := &CRDClusterConfigInfoVisitor{
		Base:    visitor.NewBase(),
		crdInfo: crdInfo,
	}
	v.SetImpl(v)
	return v
}

// VisitRoot adds CRDClusterConfigInfo to Root Extensions.
func (v *CRDClusterConfigInfoVisitor) VisitRoot(r *ast.Root) *ast.Root {
	v.errs = status.Append(v.errs, importer.AddCRDClusterConfigInfo(r, v.crdInfo))
	return r
}

// Error implements Visitor.
func (v *CRDClusterConfigInfoVisitor) Error() status.MultiError {
	return v.errs
}
