package syntax

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// FlatDirectoryValidator ensures all of the passed file paths are in a top-level directory
var FlatDirectoryValidator = &PathValidator{
	validate: func(path string) error {
		parts := strings.Split(path, string(os.PathSeparator))
		if len(parts) > 2 {
			return veterrors.IllegalSubdirectoryError{BaseDir: parts[0], SubDir: filepath.Dir(path)}
		}
		return nil
	},
}
