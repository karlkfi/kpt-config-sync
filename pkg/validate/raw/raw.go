package raw

import (
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"github.com/google/nomos/pkg/validate/raw/hydrate"
	"github.com/google/nomos/pkg/validate/raw/validate"
)

// Hierarchical performs initial validation and hydration for a structured
// hierarchical repo against the given Raw objects. Note that this will modify
// the Raw objects in-place.
func Hierarchical(objs *objects.Raw) status.MultiError {
	var errs status.MultiError
	validators := []objects.RawVisitor{
		objects.VisitAllRaw(validate.Annotations),
		objects.VisitAllRaw(validate.Labels),
		objects.VisitAllRaw(validate.IllegalKindsForHierarchical),
		objects.VisitAllRaw(validate.Namespace),
		objects.VisitAllRaw(validate.Directory),
		validate.DisallowedFields,
		validate.RemovedCRDs,
		validate.ClusterSelectors,
	}
	for _, validate := range validators {
		errs = status.Append(errs, validate(objs))
	}
	if errs != nil {
		return errs
	}

	hydrators := []objects.RawVisitor{
		hydrate.ClusterSelectors,
		hydrate.ClusterName,
	}
	for _, hydrate := range hydrators {
		errs = status.Append(errs, hydrate(objs))
	}
	return errs
}

// Unstructured performs initial validation and hydration for an unstructured
// repo against the given Raw objects. Note that this will modify the Raw
// objects in-place.
func Unstructured(objs *objects.Raw) status.MultiError {
	var errs status.MultiError
	validators := []objects.RawVisitor{
		objects.VisitAllRaw(validate.Annotations),
		objects.VisitAllRaw(validate.Labels),
		objects.VisitAllRaw(validate.IllegalKindsForUnstructured),
		objects.VisitAllRaw(validate.Namespace),
		validate.DisallowedFields,
		validate.RemovedCRDs,
		validate.ClusterSelectors,
	}
	for _, validate := range validators {
		errs = status.Append(errs, validate(objs))
	}
	if errs != nil {
		return errs
	}

	hydrators := []objects.RawVisitor{
		hydrate.ClusterSelectors,
		hydrate.ClusterName,
	}
	for _, hydrate := range hydrators {
		errs = status.Append(errs, hydrate(objs))
	}
	return errs
}
