package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInClusterregistryErrorCode is the error code for IllegalKindInClusterregistryError
const IllegalKindInClusterregistryErrorCode = "1037"

func init() {
	status.Register(IllegalKindInClusterregistryErrorCode, IllegalKindInClusterregistryError{
		Resource: role(),
	})
}

// IllegalKindInClusterregistryError reports that an object has been illegally defined in clusterregistry/
type IllegalKindInClusterregistryError struct {
	id.Resource
}

var _ id.ResourceError = &IllegalKindInClusterregistryError{}

// Error implements error
func (e IllegalKindInClusterregistryError) Error() string {
	return status.Format(e,
		"Resources of the below Kind may not be declared in %[2]s/:\n\n"+
			"%[1]s",
		id.PrintResource(e), repo.ClusterRegistryDir)
}

// Code implements Error
func (e IllegalKindInClusterregistryError) Code() string {
	return IllegalKindInClusterregistryErrorCode
}

// Resources implements ResourceError
func (e IllegalKindInClusterregistryError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
