package status

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
)

type pathErrorImpl struct {
	errorImpl errorImpl
	paths     []id.Path
}

var _ PathError = pathErrorImpl{}

// Error implements error.
func (p pathErrorImpl) Error() string {
	return format(p.errorImpl.error, formatPaths(p.paths), p.Code())
}

// Code implements Error.
func (p pathErrorImpl) Code() string {
	return p.errorImpl.Code()
}

// RelativePaths implements PathError.
func (p pathErrorImpl) RelativePaths() []id.Path {
	return p.paths
}

// ToCME implements Error.
func (p pathErrorImpl) ToCME() v1.ConfigManagementError {
	return FromPathError(p)
}
