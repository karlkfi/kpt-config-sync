package semantic

import (
	"path"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DuplicateNameValidator ensures no more than one Namespace is defined in a directory.
type DuplicateNameValidator struct {
	Objects []ast.FileObject
}

// Validate adds errors to the errorBuilder if there are multiple Namespaces defined in directories.
func (v DuplicateNameValidator) Validate(errorBuilder *multierror.Builder) {
	seenObjectNames := make(map[schema.GroupVersionKind]map[string][]ast.FileObject)

	for _, obj := range v.Objects {
		gvk := obj.GroupVersionKind()
		name := obj.Name()

		switch gvk {
		case kinds.Namespace(), kinds.ResourceQuota():
			// Namespace names are validated separately.
			// As only one ResourceQuota may currently exist in a directory, this need not be validated.
			continue
		}

		if _, found := seenObjectNames[gvk]; !found {
			seenObjectNames[gvk] = make(map[string][]ast.FileObject)
		}

		seenObjectNames[gvk][name] = append(seenObjectNames[gvk][name], obj)
	}

	// Check for object name collisions
	for _, objectsByNames := range seenObjectNames {
		// All objects have the same kind
		for name, objects := range objectsByNames {
			// All objects have the same name and kind
			sort.Slice(objects, func(i, j int) bool {
				// Sort by source file
				return path.Dir(objects[i].Source) < path.Dir(objects[j].Source)
			})

			for i := 0; i < len(objects); {
				dir := path.Dir(objects[i].Source)
				duplicates := []ast.FileObject{objects[i]}

				for j := i + 1; j < len(objects); j++ {
					if strings.HasPrefix(objects[j].Source, dir) {
						// Pick up duplicates in the same directory and child directories.
						duplicates = append(duplicates, objects[j])
					} else {
						// Since objects are sorted by paths, this guarantees that objects within a directory
						// will be contiguous. We can exit at the first non-matching source path.
						break
					}
				}

				if len(duplicates) > 1 {
					errorBuilder.Add(vet.ObjectNameCollisionError{Name: name, Duplicates: duplicates})
				}

				// Recall that len(duplicates) is always at least 1.
				// There's no need to have multiple errors when more than two objects collide.
				i += len(duplicates)
			}
		}
	}
}
