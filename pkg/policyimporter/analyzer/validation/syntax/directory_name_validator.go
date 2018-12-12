package syntax

import (
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// DirectoryNameValidator validates that directory names are valid and not reserved.
var DirectoryNameValidator = &PathValidator{
	validate: func(dir string) error {
		name := filepath.Base(dir)
		if namespaceutil.IsInvalid(name) {
			return vet.InvalidDirectoryNameError{Dir: dir}
		} else if namespaceutil.IsReserved(name) {
			return vet.ReservedDirectoryNameError{Dir: dir}
		}
		return nil
	},
}
