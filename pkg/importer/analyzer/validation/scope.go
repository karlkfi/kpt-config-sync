package validation

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
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
		p.errs = status.Append(p.errs, IllegalKindInClusterError(o))
	case discovery.UnknownScope:
		p.errs = status.Append(p.errs, UnknownObjectError(&o.FileObject))
	}

	return o
}

// VisitObject implements Visitor
func (p *Scope) VisitObject(o *ast.NamespaceObject) *ast.NamespaceObject {
	gk := o.GroupVersionKind().GroupKind()

	switch p.scoper.GetScope(gk) {
	case discovery.ClusterScope:
		if o.GroupVersionKind() != kinds.Namespace() {
			p.errs = status.Append(p.errs, syntax.IllegalKindInNamespacesError(o))
		}
	case discovery.UnknownScope:
		p.errs = status.Append(p.errs, UnknownObjectError(&o.FileObject))
	}

	return o
}

// UnknownObjectErrorCode is the error code for UnknownObjectError
const UnknownObjectErrorCode = "1021" // Impossible to create consistent example.

var unknownObjectError = status.NewErrorBuilder(UnknownObjectErrorCode)

// UnknownObjectError reports that an object declared in the repo does not have a definition in the cluster.
func UnknownObjectError(resource id.Resource) status.Error {
	return unknownObjectError.
		Sprint("No CustomResourceDefinition is defined for the resource in the cluster. " +
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.").
		BuildWithResources(resource)
}

// IllegalKindInClusterErrorCode is the error code for IllegalKindInClusterError
const IllegalKindInClusterErrorCode = "1039"

var illegalKindInClusterError = status.NewErrorBuilder(IllegalKindInClusterErrorCode)

// IllegalKindInClusterError reports that an object has been illegally defined in cluster/
func IllegalKindInClusterError(resource id.Resource) status.Error {
	return illegalKindInClusterError.
		Sprintf("Namespace-scoped configs of the below Kind must not be declared in `%s`/:", repo.ClusterDir).
		BuildWithResources(resource)
}
