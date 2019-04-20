package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalKindInNamespacesErrorCode is the error code for IllegalKindInNamespacesError
const IllegalKindInNamespacesErrorCode = "1038"

func init() {
	status.Register(IllegalKindInNamespacesErrorCode, IllegalKindInNamespacesError{
		Resource: clusterRole(),
	})
}

// IllegalKindInNamespacesError reports that an object has been illegally defined in namespaces/
type IllegalKindInNamespacesError struct {
	id.Resource
}

var _ status.ResourceError = IllegalKindInNamespacesError{}

// Error implements error
func (e IllegalKindInNamespacesError) Error() string {
	return status.Format(e,
		"Configs of the below Kind may not be declared in `%s`/:",
		repo.NamespacesDir)
}

// Code implements Error
func (e IllegalKindInNamespacesError) Code() string {
	return IllegalKindInNamespacesErrorCode
}

// Resources implements ResourceError
func (e IllegalKindInNamespacesError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalKindInNamespacesError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
