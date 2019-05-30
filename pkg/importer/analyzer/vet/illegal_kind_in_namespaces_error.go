package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInNamespacesErrorCode is the error code for IllegalKindInNamespacesError
const IllegalKindInNamespacesErrorCode = "1038"

func init() {
	status.AddExamples(IllegalKindInNamespacesErrorCode, IllegalKindInNamespacesError(
		clusterRole(),
	))
}

var illegalKindInNamespacesError = status.NewErrorBuilder(IllegalKindInNamespacesErrorCode)

// IllegalKindInNamespacesError reports that an object has been illegally defined in namespaces/
func IllegalKindInNamespacesError(resource id.Resource) status.Error {
	return illegalKindInNamespacesError.WithResources(resource).Errorf(
		"Configs of the below Kind may not be declared in `%s`/:",
		repo.NamespacesDir)
}
