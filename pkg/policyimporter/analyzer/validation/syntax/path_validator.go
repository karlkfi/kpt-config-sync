package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/status"
)

// PathValidator validates relative paths in a Nomos repository.
type PathValidator struct {
	validate func(dir nomospath.Relative) error
}

// Validate validates a list of nomospath.Relative.
func (v PathValidator) Validate(dirs []nomospath.Relative, errorBuilder *status.ErrorBuilder) {
	for _, dir := range dirs {
		errorBuilder.Add(v.validate(dir))
	}
}
