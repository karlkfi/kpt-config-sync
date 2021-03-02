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
		objects.VisitAllRaw(validate.HNCLabels),
		objects.VisitAllRaw(validate.ManagementAnnotation),
		objects.VisitAllRaw(validate.CRDName),
		validate.DisallowedFields,
		validate.RemovedCRDs,
		validate.ClusterSelectorsForHierarchical,
		validate.Repo,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	if errs != nil {
		return errs
	}

	hydrators := []objects.RawVisitor{
		hydrate.ClusterSelectors,
		hydrate.ClusterName,
		hydrate.HNCDepth,
	}
	for _, hydrator := range hydrators {
		errs = status.Append(errs, hydrator(objs))
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
		objects.VisitAllRaw(validate.ManagementAnnotation),
		objects.VisitAllRaw(validate.CRDName),
		validate.DisallowedFields,
		validate.RemovedCRDs,
		validate.ClusterSelectorsForUnstructured,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	if errs != nil {
		return errs
	}

	hydrators := []objects.RawVisitor{
		hydrate.ClusterSelectors,
		hydrate.ClusterName,
	}
	for _, hydrator := range hydrators {
		errs = status.Append(errs, hydrator(objs))
	}
	return errs
}
