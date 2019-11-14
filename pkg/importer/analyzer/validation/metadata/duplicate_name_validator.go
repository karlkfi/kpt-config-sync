package metadata

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewDuplicateNameValidator ensures the flattened config output contains no resources in the same
// config which share the same group, kind, and name.
func NewDuplicateNameValidator() ast.Visitor {
	return visitor.NewValidator(&duplicateNameValidator{})
}

// ValidateRoot ensures there are no two Namespace TreeNodes with the same name.
func (v *duplicateNameValidator) ValidateRoot(r *ast.Root) status.MultiError {
	// Normally we'd just be able to collect the Namespaces and check them. Instead, since we discard Namespaces while
	// parsing the hierarchy, we have to recursively traverse the Tree to get the full list of Namespaces.
	return CheckDuplicates(listNamespaces(r.Tree))
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

	return CheckDuplicates(resources)
}

// ValidateCluster ensures the Cluster config contains no duplicates.
func (v *duplicateNameValidator) ValidateCluster(c []*ast.ClusterObject) status.MultiError {
	resources := make([]id.Resource, len(c))
	for i, object := range c {
		resources[i] = object
	}

	return CheckDuplicates(resources)
}

// NameCollisionErrorCode is the error code for ObjectNameCollisionError
const NameCollisionErrorCode = "1029"

// nameCollisionErrorBuilder is
var nameCollisionErrorBuilder = status.NewErrorBuilder(NameCollisionErrorCode)

// NamespaceCollisionError reports multiple declared Namespaces with the same name.
func NamespaceCollisionError(name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.WithResources(duplicates...).Errorf(
		"Namespaces MUST have unique names. Found %d Namespaces named %q. Rename or merge the Namespaces to fix:",
		len(duplicates), name)
}

// NamespaceMetadataNameCollisionError reports that multiple namespace-scoped objects of the same Kind and
// namespace have the same metadata name
func NamespaceMetadataNameCollisionError(gk schema.GroupKind, namespace string, name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.WithResources(duplicates...).Errorf(
		"Namespace-scoped configs of the same Group and Kind MUST have unique names if they are in the same Namespace. "+
			"Found %d configs of GroupKind %q in Namespace %q named %q. Rename or delete the duplicates to fix:",
		len(duplicates), gk.String(), namespace, name)
}

// ClusterMetadataNameCollisionError reports that multiple cluster-scoped objects of the same Kind and
// namespace have the same metadata name
func ClusterMetadataNameCollisionError(gk schema.GroupKind, name string, duplicates ...id.Resource) status.Error {
	return nameCollisionErrorBuilder.WithResources(duplicates...).Errorf(
		"Cluster-scoped configs of the same Group and Kind MUST have unique names."+
			"Found %d configs of GroupKind %q named %q. Rename or delete the duplicates to fix:",
		len(duplicates), gk.String(), name)
}
