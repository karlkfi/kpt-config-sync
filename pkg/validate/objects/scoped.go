package objects

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
)

// ScopedVisitor is a function that validates or hydrates Scoped objects.
type ScopedVisitor func(s *Scoped) status.MultiError

// Scoped contains a collection of FileObjects that have are organized based
// upon if they are cluster-scoped or namespace-scoped.
type Scoped struct {
	Cluster               []ast.FileObject
	Namespace             []ast.FileObject
	Unknown               []ast.FileObject
	DefaultNamespace      string
	IsNamespaceReconciler bool
}

// VisitAllScoped returns a ScopedVisitor which will call the given
// ObjectVisitor on every FileObject in the Scoped objects.
func VisitAllScoped(validate ObjectVisitor) ScopedVisitor {
	return func(s *Scoped) status.MultiError {
		var errs status.MultiError
		for _, obj := range s.Cluster {
			errs = status.Append(errs, validate(obj))
		}
		for _, obj := range s.Namespace {
			errs = status.Append(errs, validate(obj))
		}
		for _, obj := range s.Unknown {
			errs = status.Append(errs, validate(obj))
		}
		return errs
	}
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
