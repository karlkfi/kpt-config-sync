package syntax

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
)

type flatNodeValidator struct {
	visitor.ValidatorBase
}

// NewFlatNodeValidator returns a Validator that ensures that system/, cluster/, and
// clusterregistry/ only define resources in top-level directories.
func NewFlatNodeValidator() ast.Visitor {
	return visitor.NewValidator(&flatNodeValidator{})
}

// ValidateSystemObject implements visitor.Validator.
func (v *flatNodeValidator) ValidateSystemObject(o *ast.SystemObject) *status.MultiError {
	return errIfNotTopLevel(o.FileObject)
}

// ValidateClusterRegistryObject implements visitor.Validator.
func (v *flatNodeValidator) ValidateClusterRegistryObject(o *ast.ClusterRegistryObject) *status.MultiError {
	return errIfNotTopLevel(o.FileObject)
}

// ValidateClusterObject implements visitor.Validator.
func (v *flatNodeValidator) ValidateClusterObject(o *ast.ClusterObject) *status.MultiError {
	return errIfNotTopLevel(o.FileObject)
}

func errIfNotTopLevel(o ast.FileObject) *status.MultiError {
	parts := o.Dir().Split()
	if !(len(parts) == 1) {
		return status.From(vet.IllegalSubdirectoryError{BaseDir: parts[0], SubDir: o.Dir()})
	}
	return nil
}
