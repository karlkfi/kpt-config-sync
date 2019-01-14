package metadata

import (
	"github.com/google/nomos/pkg/util/multierror"
)

var metaValidators []ValidatorFactory

// Register adds new metadata field validators for policy Resources.
func Register(factory ...ValidatorFactory) {
	metaValidators = append(metaValidators, factory...)
}

// Validate validates metadata fields on the given Resources.
func Validate(metas []ResourceMeta, errorBuilder *multierror.Builder) {
	for _, v := range metaValidators {
		v.New(metas).Validate(errorBuilder)
	}
}
