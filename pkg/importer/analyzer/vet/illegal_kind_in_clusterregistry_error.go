package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInClusterregistryErrorCode is the error code for IllegalKindInClusterregistryError
const IllegalKindInClusterregistryErrorCode = "1037"

func init() {
	status.AddExamples(IllegalKindInClusterregistryErrorCode, IllegalKindInClusterregistryError(role()))
}

var illegalKindInClusterregistryError = status.NewErrorBuilder(IllegalKindInClusterregistryErrorCode)

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
func IllegalKindInClusterregistryError(resource id.Resource) status.Error {
	return illegalKindInClusterregistryError.WithResources(resource).Errorf(
		"Configs of the below Kind may not be declared in `%s`/:",
		repo.ClusterRegistryDir)
}
