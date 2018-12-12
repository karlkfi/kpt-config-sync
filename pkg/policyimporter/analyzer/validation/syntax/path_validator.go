package syntax

import "github.com/google/nomos/pkg/util/multierror"

// PathValidator validates directories
type PathValidator struct {
	validate func(dir string) error
}

// Validate validates a list of directories
func (v PathValidator) Validate(dirs []string, errorBuilder *multierror.Builder) {
	for _, dir := range dirs {
		errorBuilder.Add(v.validate(dir))
	}
}
