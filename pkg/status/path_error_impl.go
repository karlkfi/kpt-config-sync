package status

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
)

type pathErrorImpl struct {
	underlying Error
	paths      []id.Path
}

var _ PathError = pathErrorImpl{}

// Error implements error.
func (p pathErrorImpl) Error() string {
	return format(p)
}

// Code implements Error.
func (p pathErrorImpl) Code() string {
	return p.underlying.Code()
}

// Body implements Error.
func (p pathErrorImpl) Body() string {
	return formatBody(p.underlying.Body(), "\n\n", formatPaths(p.paths))
}

// Errors implements MultiError.
func (p pathErrorImpl) Errors() []Error {
	return []Error{p}
}

// RelativePaths implements PathError.
func (p pathErrorImpl) RelativePaths() []id.Path {
	return p.paths
}

// Cause implements causer
func (p pathErrorImpl) Cause() error {
	return p.underlying.Cause()
}

// ToCME implements Error.
func (p pathErrorImpl) ToCME() v1.ConfigManagementError {
	return fromPathError(p)
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
