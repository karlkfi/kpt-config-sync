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

	hydrators := []objects.TreeVisitor{
		hydrate.Inheritance,
	}
	for _, hydrator := range hydrators {
		errs = status.Append(errs, hydrator(objs))
	}
	return errs
}
