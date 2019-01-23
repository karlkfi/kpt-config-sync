package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/namespaceutil"
)

// DirectoryNameValidator validates that directory names are valid and not reserved.
var DirectoryNameValidator = &PathValidator{
	validate: func(dir nomospath.Relative) error {
		name := dir.Base()
		if namespaceutil.IsInvalid(name) {
			return vet.InvalidDirectoryNameError{Dir: dir}
		} else if namespaceutil.IsReserved(name) {
			return vet.ReservedDirectoryNameError{Dir: dir}
		}
		return nil
	},
}
