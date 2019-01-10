package veterrors

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// IllegalKindInClusterregistryErrorCode is the error code for IllegalKindInClusterregistryError
const IllegalKindInClusterregistryErrorCode = "1037"

func init() {
	register(IllegalKindInClusterregistryErrorCode, nil, "")
}

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
type IllegalKindInClusterregistryError struct {
	id.Resource
}

// Error implements error
func (e IllegalKindInClusterregistryError) Error() string {
	return format(e,
		"Resources of the below Kind may not be declared in %[2]s/:\n\n"+
			"%[1]s",
		id.PrintResource(e), repo.ClusterRegistryDir)
}

// Code implements Error
func (e IllegalKindInClusterregistryError) Code() string {
	return IllegalKindInClusterregistryErrorCode
}
