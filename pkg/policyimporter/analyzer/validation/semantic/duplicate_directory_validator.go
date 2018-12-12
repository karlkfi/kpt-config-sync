package semantic

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
)

// DuplicateDirectoryValidator ensures no two directories share the same base name.
type DuplicateDirectoryValidator struct {
	// The set of distinct full directory paths in a Nomos repo relative to Nomos root
	// OS-agnostic
	Dirs []string
}

// Validate implements Validator
func (v DuplicateDirectoryValidator) Validate(errorBuilder *multierror.Builder) {
	// a map from the set of base paths to the full set of file paths which have that base path
	// so if bar/foo and qux/foo exist, then we would have the entry foo: [bar/foo, qux/foo]
	fullPaths := make(map[string]map[string]bool, len(v.Dirs))

	for _, fullPath := range v.Dirs {
		parts := strings.Split(fullPath, string(os.PathSeparator))
		for i := range parts {
			path := filepath.Join(parts[0 : i+1]...)
			base := filepath.Base(path)

			if _, found := fullPaths[base]; found {
				fullPaths[base][path] = true
			} else {
				fullPaths[base] = map[string]bool{path: true}
			}
		}
	}

	for _, fullPaths := range fullPaths {
		if len(fullPaths) > 1 {
			// More than one unique path had the same base path
			var duplicates []string
			for fullPath := range fullPaths {
				duplicates = append(duplicates, fullPath)
			}
			errorBuilder.Add(vet.DuplicateDirectoryNameError{Duplicates: duplicates})
		}
	}
}
