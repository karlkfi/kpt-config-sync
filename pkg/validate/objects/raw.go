package objects

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// RawVisitor is a function that validates or hydrates Raw objects.
type RawVisitor func(r *Raw) status.MultiError

// ObjectVisitor is a function that validates a single FileObject at a time.
type ObjectVisitor func(obj ast.FileObject) status.Error

// Raw contains a collection of FileObjects that have just been parsed from a
// Git repo for a cluster.
type Raw struct {
	ClusterName      string
	Objects          []ast.FileObject
	PreviousCRDs     []*v1beta1.CustomResourceDefinition
	BuildScoper      utildiscovery.BuildScoperFunc
	ErrOnUnknownKind bool
}

// Scoped builds a Scoped collection of objects from the Raw objects.
func (r *Raw) Scoped() (*Scoped, status.MultiError) {
	scoper, errs := r.BuildScoper(r.PreviousCRDs, r.Objects)
	if errs != nil {
		return nil, errs
	}
	scoped := &Scoped{}
	for _, obj := range r.Objects {
		s, err := scoper.GetObjectScope(obj)
		if err != nil {
			if r.ErrOnUnknownKind {
				errs = status.Append(errs, err)
			} else {
				glog.V(6).Infof("ignoring error: %v", err)
			}
		}

		switch s {
		case utildiscovery.ClusterScope:
			scoped.Cluster = append(scoped.Cluster, obj)
		case utildiscovery.NamespaceScope:
			scoped.Namespace = append(scoped.Namespace, obj)
		case utildiscovery.UnknownScope:
			scoped.Unknown = append(scoped.Unknown, obj)
		default:
			errs = status.Append(errs, status.InternalErrorf("unrecognized discovery scope: %s", s))
		}
	}
	if errs != nil {
		return nil, errs
	}
	return scoped, nil
}

// VisitAllRaw returns a RawVisitor which will call the given ObjectVisitor on
// every FileObject in the Raw objects.
func VisitAllRaw(visit ObjectVisitor) RawVisitor {
	return func(r *Raw) status.MultiError {
		var errs status.MultiError
		for _, obj := range r.Objects {
			errs = status.Append(errs, visit(obj))
		}
		return errs
	}
}
