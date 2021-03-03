package scoped

import (
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"github.com/google/nomos/pkg/validate/scoped/hydrate"
	"github.com/google/nomos/pkg/validate/scoped/validate"
)

// Hierarchical performs the second round of validation and hydration for a
// structured hierarchical repo against the given Scoped objects. Note that this
// will modify the Scoped objects in-place.
func Hierarchical(objs *objects.Scoped) status.MultiError {
	var errs status.MultiError
	// See the note about ordering raw.Hierarchical().
	validators := []objects.ScopedVisitor{
		objects.VisitClusterScoped(validate.ClusterScoped),
		objects.VisitNamespaceScoped(validate.NamespaceScoped),
		validate.NamespaceSelectors,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	if errs != nil {
		return errs
	}

	hydrators := []objects.ScopedVisitor{}
	for _, hydrator := range hydrators {
		errs = status.Append(errs, hydrator(objs))
	}
	return errs
}

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

	// See the note about ordering raw.Hierarchical().
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
