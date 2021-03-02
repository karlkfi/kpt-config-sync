package hydrate

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/validate/hydrate/common"
	"github.com/google/nomos/pkg/validate/parsed"
)

// FlatHydrators returns the list of hydrators to process a flat repo.
func FlatHydrators(policyDir cmpath.Relative) []parsed.FlatHydrator {
	return []parsed.FlatHydrator{
		common.FilepathFlatHydrator(policyDir),
	}
}

// HierarchicalHydrators returns the list of hydrators to process a hierarchical
// repo.
func HierarchicalHydrators(policyDir cmpath.Relative) []parsed.TreeHydrator {
	return []parsed.TreeHydrator{
		common.FilepathTreeHydrator(policyDir),
	}
}
