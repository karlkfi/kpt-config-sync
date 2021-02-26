package prehydrate

import (
	"github.com/google/nomos/pkg/validate/parsed"
	"github.com/google/nomos/pkg/validate/prehydrate/hierarchical"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// Config contains parameters for prehydration validation.
type Config struct {
	PreviousCRDs []*v1beta1.CustomResourceDefinition
	CurrentCRDs  []*v1beta1.CustomResourceDefinition
}

// FlatValidators returns the list of visitors to validate a flat repo
// pre-hydration.
func FlatValidators(config Config) []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{}
}

// HierarchicalValidators returns the list of visitors to validate a
// hierarchical repo pre-hydration.
func HierarchicalValidators(config Config) []parsed.ValidatorFunc {
	return []parsed.ValidatorFunc{
		hierarchical.NamespaceSelectorValidator(),
		hierarchical.InheritanceValidator(),
	}
}
