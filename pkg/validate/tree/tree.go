package tree

import (
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"github.com/google/nomos/pkg/validate/tree/hydrate"
	"github.com/google/nomos/pkg/validate/tree/validate"
)

// Hierarchical performs validation and hydration for a structured hierarchical
// repo against the given Tree objects. Note that this will modify the Tree
// objects in-place.
func Hierarchical(objs *objects.Tree) status.MultiError {
	var errs status.MultiError
	// See the note about ordering in raw.Hierarchical().
	validators := []objects.TreeVisitor{
		validate.HierarchyConfig,
		validate.Inheritance,
		validate.NamespaceSelector,
	}
	for _, validator := range validators {
		errs = status.Append(errs, validator(objs))
	}
	if errs != nil {
		return errs
	}

	// We perform inheritance first so that we copy all abstract objects into
	// their potential namespaces, and then we perform namespace selection to
	// filter out the copies which are not selected.
	hydrators := []objects.TreeVisitor{
		hydrate.Inheritance,
		hydrate.NamespaceSelectors,
	}
	for _, hydrator := range hydrators {
		errs = status.Append(errs, hydrator(objs))
	}
	return errs
}
