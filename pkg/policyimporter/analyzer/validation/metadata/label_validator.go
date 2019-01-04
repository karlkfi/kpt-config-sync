package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// LabelValidatorFactory validates the labels in a ast.FileObject
var LabelValidatorFactory = ValidatorFactory{
	fn: func(meta ResourceMeta) error {
		var errors []string
		for l := range meta.MetaObject().GetLabels() {
			if v1alpha1.HasNomosPrefix(l) {
				errors = append(errors, l)
			}
		}
		if errors != nil {
			return veterrors.IllegalLabelDefinitionError{ResourceID: meta, Labels: errors}
		}
		return nil
	},
}
