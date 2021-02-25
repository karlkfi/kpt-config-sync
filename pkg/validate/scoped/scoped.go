package scoped

import (
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"github.com/google/nomos/pkg/validate/scoped/hydrate"
	"github.com/google/nomos/pkg/validate/scoped/validate"
)

// Unstructured performs the second round of validation and hydration for an
// unstructured repo against the given Scoped objects. Note that this will
// modify the Scoped objects in-place.
func Unstructured(objs *objects.Scoped) status.MultiError {
	var errs status.MultiError
	var validateClusterScoped objects.ObjectVisitor
	if objs.IsNamespaceReconciler {
		validateClusterScoped = validate.ClusterScopedForNamespaceReconciler
	} else {
		validateClusterScoped = validate.ClusterScoped
	}

	validators := []objects.ScopedVisitor{
		objects.VisitClusterScoped(validateClusterScoped),
		objects.VisitNamespaceScoped(validate.NamespaceScoped),
		validate.NamespaceSelectors,
	}
	for _, validate := range validators {
		errs = status.Append(errs, validate(objs))
	}
	if errs != nil {
		return errs
	}

	hydrators := []objects.ScopedVisitor{
		hydrate.NamespaceSelectors,
	}
	for _, hydrate := range hydrators {
		errs = status.Append(errs, hydrate(objs))
	}
	return errs
}
