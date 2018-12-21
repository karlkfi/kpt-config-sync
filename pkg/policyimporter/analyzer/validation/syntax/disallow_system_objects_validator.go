package syntax

import (
	"path"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DisallowSystemObjectsValidator validates that the resources which may appear in system/ and nowhere
// else only appear in system/.
var DisallowSystemObjectsValidator = &FileObjectValidator{
	validate: func(fileObject ast.FileObject) error {
		if IsSystemOnly(fileObject.GroupVersionKind()) && !isInSystemDir(fileObject) {
			return vet.IllegalSystemResourcePlacementError{Object: fileObject}
		}
		return nil
	},
}

// isInSystemDir returns true if the Resource is currently placed in system/.
func isInSystemDir(o ast.FileObject) bool {
	return strings.HasPrefix(path.Dir(o.Source), repo.SystemDir)
}

// IsSystemOnly returns true if the GVK is only allowed in the system/ directory.
// It returns true iff the object is allowed in system/, but no other directories.
func IsSystemOnly(gvk schema.GroupVersionKind) bool {
	switch gvk {
	case kinds.Repo(), kinds.Sync():
		return true
	default:
		return false
	}
}
