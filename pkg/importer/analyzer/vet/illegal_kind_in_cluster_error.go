package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInClusterErrorCode is the error code for IllegalKindInClusterError
const IllegalKindInClusterErrorCode = "1039"

func init() {
	status.Register(IllegalKindInClusterErrorCode, IllegalKindInClusterError{
		Resource: role(),
	})
}

// IllegalKindInClusterError reports that an object has been illegally defined in cluster/
type IllegalKindInClusterError struct {
	id.Resource
}

var _ status.ResourceError = &IllegalKindInClusterError{}

// Error implements error
func (e IllegalKindInClusterError) Error() string {
	return status.Format(e,
		"Namespace-scoped configs of the below Kind must not be declared in `%[2]s`/:\n\n"+
			"%[1]s",
		id.PrintResource(e), repo.ClusterDir)
}

// Code implements Error
func (e IllegalKindInClusterError) Code() string {
	return IllegalKindInClusterErrorCode
}

// Resources implements ResourceError
func (e IllegalKindInClusterError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
