package syntax

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
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
			return IllegalKindInClusterregistryError(object)
		}
		return nil
	})
}

// IllegalKindInClusterregistryErrorCode is the error code for IllegalKindInClusterregistryError
const IllegalKindInClusterregistryErrorCode = "1037"

var illegalKindInClusterregistryError = status.NewErrorBuilder(IllegalKindInClusterregistryErrorCode)

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
func IllegalKindInClusterregistryError(resource id.Resource) status.Error {
	return illegalKindInClusterregistryError.WithResources(resource).Errorf(
		"Configs of the below Kind may not be declared in `%s`/:",
		repo.ClusterRegistryDir)
}
