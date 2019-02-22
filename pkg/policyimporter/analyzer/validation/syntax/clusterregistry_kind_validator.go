package syntax

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// NewClusterRegistryKindValidator ensures only the allowed set of types appear in clusterregistry/
func NewClusterRegistryKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterRegistryObjectValidator(func(object *ast.ClusterRegistryObject) error {
		switch object.Object.(type) {
		case *v1.ClusterSelector:
		case *clusterregistry.Cluster:
		default:
			return vet.IllegalKindInClusterregistryError{Resource: object}
		}
		return nil
	})
}
