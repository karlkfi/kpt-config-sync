package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInClusterErrorCode is the error code for IllegalKindInClusterError
const IllegalKindInClusterErrorCode = "1039"

func init() {
	status.AddExamples(IllegalKindInClusterErrorCode, IllegalKindInClusterError(
		role(),
	))
}

var illegalKindInClusterError = status.NewErrorBuilder(IllegalKindInClusterErrorCode)

// IllegalKindInClusterError reports that an object has been illegally defined in cluster/
func IllegalKindInClusterError(resource id.Resource) status.Error {
	return illegalKindInClusterError.WithResources(resource).Errorf(
		"Namespace-scoped configs of the below Kind must not be declared in `%s`/:",
		repo.ClusterDir)
}
