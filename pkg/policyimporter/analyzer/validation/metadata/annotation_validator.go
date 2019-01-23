package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

func init() {
	Register(AnnotationValidatorFactory)
}

// AnnotationValidatorFactory returns errors
var AnnotationValidatorFactory = SyntaxValidatorFactory{
	fn: func(meta ResourceMeta) error {
		var errors []string
		for a := range meta.MetaObject().GetAnnotations() {
			if !v1alpha1.IsInputAnnotation(a) && v1alpha1.HasNomosPrefix(a) {
				errors = append(errors, a)
			}
		}
		if errors != nil {
			return vet.IllegalAnnotationDefinitionError{Resource: meta, Annotations: errors}
		}
		return nil
	},
}
