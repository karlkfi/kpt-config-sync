package status

import (
	"path/filepath"
	"sort"
	"strings"

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

// Errors implements MultiError.
func (p pathErrorImpl) Errors() []Error {
	return []Error{p}
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

func formatPaths(paths []id.Path) string {
	pathStrs := make([]string, len(paths))
	for i, path := range paths {
		pathStrs[i] = "path: " + path.OSPath()
		if filepath.Ext(path.OSPath()) == "" {
			// Assume paths without extensions are directories. We don't support files without extensions,
			// so for now this is a safe assumption.
			pathStrs[i] += "/"
		}
	}
	// Ensure deterministic path printing order.
	sort.Strings(pathStrs)
	return strings.Join(pathStrs, "\n")
}
