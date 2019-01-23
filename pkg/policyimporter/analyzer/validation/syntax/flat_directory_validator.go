package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
)

// FlatDirectoryValidator ensures all of the passed file paths are in a top-level directory
var FlatDirectoryValidator = &PathValidator{
	validate: func(path nomospath.Relative) error {
		parts := path.Split()
		if len(parts) > 2 {
			return vet.IllegalSubdirectoryError{BaseDir: parts[0], SubDir: path.Dir()}
		}
		return nil
	},
}
