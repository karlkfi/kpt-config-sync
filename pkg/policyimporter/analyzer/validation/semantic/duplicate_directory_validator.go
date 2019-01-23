package semantic

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

// DuplicateDirectoryValidator ensures no two directories share the same base name.
type DuplicateDirectoryValidator struct {
	// The set of distinct full directory paths in a Nomos repo relative to Nomos root
	// OS-agnostic
	Dirs []nomospath.Relative
}

// Validate implements Validator
func (v DuplicateDirectoryValidator) Validate(errorBuilder *multierror.Builder) {
	// a map from the set of base paths to the full set of file paths which have that base path
	// so if bar/foo and qux/foo exist, then we would have the entry foo: [bar/foo, qux/foo]
	fullPaths := make(map[string]map[nomospath.Relative]bool, len(v.Dirs))

	for _, fullPath := range v.Dirs {
		for curDir := fullPath; !curDir.IsRoot(); {
			base := curDir.Base()
			if _, found := fullPaths[base]; found {
				fullPaths[base][curDir] = true
			} else {
				fullPaths[base] = map[nomospath.Relative]bool{curDir: true}
			}
			curDir = curDir.Dir()
		}
	}

	for _, duplicatesMap := range fullPaths {
		if len(duplicatesMap) > 1 {
			// More than one unique path had the same base path
			var duplicates []nomospath.Relative
			for duplicate := range duplicatesMap {
				duplicates = append(duplicates, duplicate)
			}
			errorBuilder.Add(vet.DuplicateDirectoryNameError{Duplicates: duplicates})
		}
	}
}
