package objects

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// ScopedVisitor is a function that validates or hydrates Scoped objects.
type ScopedVisitor func(s *Scoped) status.MultiError

// Scoped contains a collection of FileObjects that are organized based upon if
// they are cluster-scoped or namespace-scoped.
type Scoped struct {
	Cluster               []ast.FileObject
	Namespace             []ast.FileObject
	Unknown               []ast.FileObject
	DefaultNamespace      string
	IsNamespaceReconciler bool
}

// Objects returns all FileObjects in the Scoped collection.
func (s *Scoped) Objects() []ast.FileObject {
	objs := append(s.Cluster, s.Unknown...)
	return append(objs, s.Namespace...)
}

// VisitClusterScoped returns a ScopedVisitor which will call the given
// ObjectVisitor on all cluster-scoped FileObjects in the Scoped objects.
func VisitClusterScoped(validate ObjectVisitor) ScopedVisitor {
	return func(s *Scoped) status.MultiError {
		var errs status.MultiError
		for _, obj := range s.Cluster {
			errs = status.Append(errs, validate(obj))
		}
		return errs
	}
}

// VisitNamespaceScoped returns a ScopedVisitor which will call the given
// ObjectVisitor on all namespace-scoped FileObjects in the Scoped objects.
func VisitNamespaceScoped(validate ObjectVisitor) ScopedVisitor {
	return func(s *Scoped) status.MultiError {
		var errs status.MultiError
		for _, obj := range s.Namespace {
			errs = status.Append(errs, validate(obj))
		}
		return errs
	}
}
