package prehydrate

import (
	"github.com/google/nomos/pkg/validate/parsed"
	"github.com/google/nomos/pkg/validate/prehydrate/hierarchical"
)

// FlatValidators returns the list of visitors to validate a flat repo
// pre-hydration.
func FlatValidators() []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{}
}

// HierarchicalValidators returns the list of visitors to validate a
// hierarchical repo pre-hydration.
func HierarchicalValidators() []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{
		hierarchical.NamespaceDirectoryValidator(),
		hierarchical.ObjectDirectoryValidator(),
		hierarchical.DirectoryNameValidator(),
		hierarchical.NamespaceSelectorValidator(),
	}
}
