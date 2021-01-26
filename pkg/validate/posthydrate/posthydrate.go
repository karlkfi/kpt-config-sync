package posthydrate

import (
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/validate/parsed"
	"github.com/google/nomos/pkg/validate/posthydrate/hierarchical"
)

// FlatValidators returns the list of visitors to validate a flat repo
// post-hydration.
func FlatValidators() []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{}
}

// HierarchicalValidators returns the list of visitors to validate a
// hierarchical repo post-hydration.
func HierarchicalValidators() []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{
		hierarchical.RepoVersionValidator(),
		hierarchical.SingletonValidator(kinds.Repo()),
		hierarchical.TreeNodeSingletonValidator(kinds.Namespace()),
		hierarchical.HierarchyConfigValidator(),
	}
}
