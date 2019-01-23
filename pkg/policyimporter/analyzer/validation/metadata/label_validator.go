package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

func init() {
	Register(LabelValidatorFactory)
}

// LabelValidatorFactory validates the labels declared in metadata
var LabelValidatorFactory = SyntaxValidatorFactory{
	fn: func(meta ResourceMeta) error {
		var errors []string
		for l := range meta.MetaObject().GetLabels() {
			if v1alpha1.HasNomosPrefix(l) {
				errors = append(errors, l)
			}
		}
		if errors != nil {
			return vet.IllegalLabelDefinitionError{Resource: meta, Labels: errors}
		}
		return nil
	},
}
