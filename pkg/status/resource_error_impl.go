package status

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
)

type resourceErrorImpl struct {
	errorImpl errorImpl
	resources []id.Resource
}

var _ ResourceError = resourceErrorImpl{}

// Error implements error.
func (p resourceErrorImpl) Error() string {
	return format(p.errorImpl.error, formatResources(p.resources), p.Code())
}

// Code implements Error.
func (p resourceErrorImpl) Code() string {
	return p.errorImpl.Code()
}

// Resources implements ResourceError.
func (p resourceErrorImpl) Resources() []id.Resource {
	return p.resources
}

// ToCME implements Error.
func (p resourceErrorImpl) ToCME() v1.ConfigManagementError {
	return FromResourceError(p)
}
