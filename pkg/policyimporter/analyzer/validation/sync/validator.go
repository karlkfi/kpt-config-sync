package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type groupKind struct {
	group string
	kind  string
}

// Identifies a Group/Kind definition in a Sync.
// This is not unique if the same Sync Resource defines the multiple of the same Group/Kind.
type kindSync struct {
	// sync is the Sync which defined the Kind.
	sync ast.FileObject
	// gvk is the Group/Version/Kind which the Sync defined
	gvk schema.GroupVersionKind
	// hierarchy is the hierarchy mode which the Sync defined for the Kind.
	hierarchy v1alpha1.HierarchyModeType
}

// validator validates Kind declarations in Sync Resources
type validator struct {
	validate func(sync kindSync) error
}

// Validate adds errors for each unsupported Kind defined in a Sync.
// It abstracts out the deeply-nested logic for extracting every Kind defined in every Sync.
func (v validator) Validate(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	for _, sync := range kindSyncs(objects) {
		errorBuilder.Add(v.validate(sync))
	}
}
