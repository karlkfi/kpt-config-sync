package syntax

import (
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// DirectoryNameValidator validates that directory names are valid and not reserved.
var DirectoryNameValidator = &PathValidator{
	validate: func(dir string) error {
		name := filepath.Base(dir)
		if namespaceutil.IsInvalid(name) {
			return veterrors.InvalidDirectoryNameError{Dir: dir}
		} else if namespaceutil.IsReserved(name) {
			return veterrors.ReservedDirectoryNameError{Dir: dir}
		}
		return nil
	},
}
