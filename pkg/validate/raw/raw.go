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
	// Note that the ordering here and in all other collections of validators is
	// somewhat arbitrary. We always run all validators in a collection before
	// exiting with any errors. We do put more "fundamental" validation checks
	// first so that their errors will appear first. For example, we check for
	// illegal objects first before we check for an illegal namespace on that
	// object. That way the first error in the list is more likely the real
	// problem (eg they need to remove the object rather than fixing its
	// namespace).
	validators := []objects.RawVisitor{
		objects.VisitAllRaw(validate.Annotations),
		objects.VisitAllRaw(validate.Labels),
		objects.VisitAllRaw(validate.IllegalKindsForHierarchical),
		objects.VisitAllRaw(validate.DeprecatedKinds),
		objects.VisitAllRaw(validate.Name),
		objects.VisitAllRaw(validate.Namespace),
		objects.VisitAllRaw(validate.Directory),
		objects.VisitAllRaw(validate.HNCLabels),
		objects.VisitAllRaw(validate.ManagementAnnotation),
		objects.VisitAllRaw(validate.IllegalCRD),
		objects.VisitAllRaw(validate.CRDName),
		objects.VisitAllRaw(validate.RepoSync),
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

	// First we annotate all objects with their declared fields. It is crucial
	// that we do this step before any other hydration so that we capture the
	// object exactly as it is declared in Git. Next we set missing namespaces on
	// objects in namespace directories since cluster selection relies on
	// namespace if a namespace gets filtered out. Then we perform cluster
	// selection so that we can filter out irrelevant objects before trying to
	// modify them.
	hydrators := []objects.RawVisitor{
		hydrate.DeclaredFields,
		hydrate.ObjectNamespaces,
		hydrate.ClusterSelectors,
		hydrate.ClusterName,
		hydrate.Filepath,
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
	// See the note about ordering above in Hierarchical().
	validators := []objects.RawVisitor{
		objects.VisitAllRaw(validate.Annotations),
		objects.VisitAllRaw(validate.Labels),
		objects.VisitAllRaw(validate.IllegalKindsForUnstructured),
		objects.VisitAllRaw(validate.DeprecatedKinds),
		objects.VisitAllRaw(validate.Name),
		objects.VisitAllRaw(validate.Namespace),
		objects.VisitAllRaw(validate.ManagementAnnotation),
		objects.VisitAllRaw(validate.IllegalCRD),
		objects.VisitAllRaw(validate.CRDName),
		objects.VisitAllRaw(validate.RepoSync),
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

	// First we annotate all objects with their declared fields. It is crucial
	// that we do this step before any other hydration so that we capture the
	// object exactly as it is declared in Git. Then we perform cluster selection
	// so that we can filter out irrelevant objects before trying to modify them.
	hydrators := []objects.RawVisitor{
		hydrate.DeclaredFields,
		hydrate.ClusterSelectors,
		hydrate.ClusterName,
		hydrate.Filepath,
	}
	for _, hydrator := range hydrators {
		errs = status.Append(errs, hydrator(objs))
	}
	return errs
}
