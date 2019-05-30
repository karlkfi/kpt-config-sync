package syntax

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// NewClusterRegistryKindValidator ensures only the allowed set of types appear in clusterregistry/
func NewClusterRegistryKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterRegistryObjectValidator(func(object *ast.ClusterRegistryObject) status.MultiError {
		switch object.Object.(type) {
		case *v1.ClusterSelector:
		case *clusterregistry.Cluster:
		default:
			return status.From(vet.IllegalKindInClusterregistryError(object))
		}
		return nil
	})
}
