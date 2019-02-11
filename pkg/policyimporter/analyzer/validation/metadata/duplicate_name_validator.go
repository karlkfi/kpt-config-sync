package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/util/multierror"
)

type groupKindName struct {
	group string
	kind  string
	name  string
}

type duplciateNameValidator struct {
	visitor.ValidatorBase
}

// NewDuplicateNameValidator ensures the flattened policy output contains no resources in the same
// policy node which share the same group, kind, and name.
func NewDuplicateNameValidator() ast.Visitor {
	return visitor.NewValidator(&duplciateNameValidator{})
}

func checkDuplicates(objects []id.Resource) []error {
	duplicateMap := make(map[groupKindName][]id.Resource)

	for _, o := range objects {
		gkn := groupKindName{
			group: o.GroupVersionKind().Group,
			kind:  o.GroupVersionKind().Kind,
			name:  o.Name(),
		}
		duplicateMap[gkn] = append(duplicateMap[gkn], o)
	}

	var errs []error
	for _, duplicates := range duplicateMap {
		if len(duplicates) > 1 {
			errs = append(errs, vet.MetadataNameCollisionError{Duplicates: duplicates})
		}
	}
	return errs
}

// ValidateTreeNode ensures Namespace policy nodes contain no duplicates.
func (v *duplciateNameValidator) ValidateTreeNode(n *ast.TreeNode) error {
	resources := make([]id.Resource, len(n.Objects))
	for i, object := range n.Objects {
		resources[i] = object
	}

	return multierror.From(checkDuplicates(resources))
}

// ValidateCluster ensures the Cluster policy node contains no duplicates.
func (v *duplciateNameValidator) ValidateCluster(c *ast.Cluster) error {
	resources := make([]id.Resource, len(c.Objects))
	for i, object := range c.Objects {
		resources[i] = object
	}

	return multierror.From(checkDuplicates(resources))
}
