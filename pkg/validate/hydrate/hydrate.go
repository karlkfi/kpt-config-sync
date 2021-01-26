package hydrate

import (
	"github.com/google/nomos/pkg/validate/hydrate/hierarchical"
	"github.com/google/nomos/pkg/validate/parsed"
)

// HierarchicalHydrators returns the list of hydrators to process a hierarchical
// repo.
func HierarchicalHydrators() []parsed.TreeHydrator {
	return []parsed.TreeHydrator{
		hierarchical.NewInheritanceHydrator(),
	}
}
