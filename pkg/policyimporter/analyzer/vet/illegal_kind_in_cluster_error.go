package vet

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1/repo"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInClusterErrorCode is the error code for IllegalKindInClusterError
const IllegalKindInClusterErrorCode = "1039"

func init() {
	register(IllegalKindInClusterErrorCode, nil, "")
}

// IllegalKindInClusterError reports that an object has been illegally defined in cluster/
type IllegalKindInClusterError struct {
	id.Resource
}

// Error implements error
func (e IllegalKindInClusterError) Error() string {
	return status.Format(e,
		"Namespace scoped resources of the below Kind must not be declared in %[2]s/:\n\n"+
			"%[1]s",
		id.PrintResource(e), repo.ClusterDir)
}

// Code implements Error
func (e IllegalKindInClusterError) Code() string {
	return IllegalKindInClusterErrorCode
}
