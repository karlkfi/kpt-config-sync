package syntax

import (
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime"
)

// ObjectValidator validates local state of a single runtime.Object
type ObjectValidator struct {
	validate func(source string, object runtime.Object) error
}

// Validate validates a slice of Infos.
func (v ObjectValidator) Validate(objects map[runtime.Object]string, errorBuilder *multierror.Builder) {
	for object, source := range objects {
		errorBuilder.Add(v.validate(source, object))
	}
}
