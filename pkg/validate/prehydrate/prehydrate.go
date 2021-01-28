package prehydrate

import (
	"github.com/google/nomos/pkg/validate/parsed"
	"github.com/google/nomos/pkg/validate/prehydrate/common"
	"github.com/google/nomos/pkg/validate/prehydrate/hierarchical"
	"github.com/google/nomos/pkg/validate/prehydrate/hnc"
)

// FlatValidators returns the list of visitors to validate a flat repo
// pre-hydration.
func FlatValidators() []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{
		common.AnnotationValidator(),
		common.LabelValidator(),
	}
}

// HierarchicalValidators returns the list of visitors to validate a
// hierarchical repo pre-hydration.
func HierarchicalValidators() []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{
		common.AnnotationValidator(),
		common.LabelValidator(),
		hierarchical.NamespaceDirectoryValidator(),
		hierarchical.ObjectDirectoryValidator(),
		hierarchical.DirectoryNameValidator(),
		hierarchical.NamespaceSelectorValidator(),
		hnc.DepthLabelValidator(),
	}
}
