package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// NewClusterRegistryKindValidator ensures only the allowed set of types appear in clusterregistry/
func NewClusterRegistryKindValidator() *visitor.ValidatorVisitor {
	return visitor.NewClusterRegistryObjectValidator(func(object *ast.ClusterRegistryObject) status.MultiError {
		gvk := object.GroupVersionKind()
		if gvk == kinds.ClusterSelector() || gvk == kinds.Cluster() {
			// Only ClusterSelectors and Clusters are allowed in clusterregistry/.
			return nil
		}
		return IllegalKindInClusterregistryError(object)
	})
}

// IllegalKindInClusterregistryErrorCode is the error code for IllegalKindInClusterregistryError
const IllegalKindInClusterregistryErrorCode = "1037"

var illegalKindInClusterregistryError = status.NewErrorBuilder(IllegalKindInClusterregistryErrorCode)

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
func IllegalKindInClusterregistryError(resource id.Resource) status.Error {
	return illegalKindInClusterregistryError.
		Sprintf("Configs of the below Kind may not be declared in `%s`/:", repo.ClusterRegistryDir).
		BuildWithResources(resource)
}
