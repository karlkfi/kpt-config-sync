package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

type groupKindNamespaceName struct {
	group     string
	kind      string
	namespace string
	name      string
}

type duplicateNameValidator struct {
	visitor.ValidatorBase
}

type duplicateError func(...id.Resource) status.Error

// NewDuplicateNameValidator ensures the flattened config output contains no resources in the same
// config which share the same group, kind, and name.
func NewDuplicateNameValidator() ast.Visitor {
	return visitor.NewValidator(&duplicateNameValidator{})
}

// CheckDuplicates returns an error if it detects multiple objects with the same Group, Kind,
// metadata.namespace, and metadata.name.
func CheckDuplicates(objects []id.Resource, errorType duplicateError) status.MultiError {
	duplicateMap := make(map[groupKindNamespaceName][]id.Resource)

	for _, o := range objects {
		gknn := groupKindNamespaceName{
			group:     o.GroupVersionKind().Group,
			kind:      o.GroupVersionKind().Kind,
			namespace: o.Namespace(),
			name:      o.Name(),
		}
		duplicateMap[gknn] = append(duplicateMap[gknn], o)
	}

	var errs status.MultiError
	for _, duplicates := range duplicateMap {
		if len(duplicates) > 1 {
			errs = status.Append(errs, errorType(duplicates...))
		}
	}
	return errs
}

// ValidateTreeNode ensures Namespace configs contain no duplicates.
func (v *duplicateNameValidator) ValidateTreeNode(n *ast.TreeNode) status.MultiError {
	if n.Type != node.Namespace {
		return nil
	}
	resources := make([]id.Resource, len(n.Objects))
	for i, object := range n.Objects {
		resources[i] = object
	}

	return CheckDuplicates(resources, vet.NamespaceMetadataNameCollisionError)
}

// ValidateCluster ensures the Cluster config contains no duplicates.
func (v *duplicateNameValidator) ValidateCluster(c []*ast.ClusterObject) status.MultiError {
	resources := make([]id.Resource, len(c))
	for i, object := range c {
		resources[i] = object
	}

	return CheckDuplicates(resources, vet.ClusterMetadataNameCollisionError)
}
