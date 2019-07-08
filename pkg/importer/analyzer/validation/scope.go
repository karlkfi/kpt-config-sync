package validation

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// Scope runs after all transforms have completed.  This will verify that the final state of
// the tree meets various conditions before we set it on the API server.
type Scope struct {
	*visitor.Base
	errs   status.MultiError
	scoper discovery.Scoper
}

var _ ast.Visitor = &Scope{}

// NewScope returns a validator that checks if objects are in the correct scope in terms of namespace
// vs cluster.
// resourceLists is the list of supported types from the discovery client.
func NewScope() *Scope {
	pv := &Scope{
		Base: visitor.NewBase(),
	}
	pv.SetImpl(pv)
	return pv
}

// Error returns any errors encountered during processing
func (p *Scope) Error() status.MultiError {
	return p.errs
}

// VisitRoot implement ast.Visitor.
func (p *Scope) VisitRoot(r *ast.Root) *ast.Root {
	var err error
	p.scoper, err = discovery.GetScoper(r)
	p.errs = status.Append(p.errs, err)
	return p.Base.VisitRoot(r)
}

// VisitClusterObject implements Visitor
func (p *Scope) VisitClusterObject(o *ast.ClusterObject) *ast.ClusterObject {
	gk := o.GroupVersionKind().GroupKind()

	switch p.scoper.GetScope(gk) {
	case discovery.NamespaceScope:
		p.errs = status.Append(p.errs, vet.IllegalKindInClusterError(o))
	case discovery.UnknownScope:
		p.errs = status.Append(p.errs, vet.UnknownObjectError(&o.FileObject))
	}

	return o
}

// VisitObject implements Visitor
func (p *Scope) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	gk := o.GroupVersionKind().GroupKind()

	switch p.scoper.GetScope(gk) {
	case discovery.ClusterScope:
		p.errs = status.Append(p.errs, vet.IllegalKindInNamespacesError(o))
	case discovery.UnknownScope:
		p.errs = status.Append(p.errs, vet.UnknownObjectError(&o.FileObject))
	}

	return o
}
