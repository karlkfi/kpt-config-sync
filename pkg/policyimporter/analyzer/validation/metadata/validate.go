package metadata

import (
	"github.com/google/nomos/pkg/util/multierror"
)

// Validate validates metadata fields on the given Resources. These validations are
// Group/Version/Kind-independent.
func Validate(metas []ResourceMeta, errorBuilder *multierror.Builder) {
	AnnotationValidatorFactory.New(metas).Validate(errorBuilder)
	LabelValidatorFactory.New(metas).Validate(errorBuilder)
	NamespaceValidatorFactory.New(metas).Validate(errorBuilder)
	NameValidatorFactory.New(metas).Validate(errorBuilder)
	DuplicateNameValidatorFactory{}.New(metas).Validate(errorBuilder)
}
