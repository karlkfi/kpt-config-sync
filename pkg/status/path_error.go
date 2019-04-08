package status

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/importer/id"
)

// PathErrorCode is the error code for a generic PathError.
const PathErrorCode = "2001"

func init() {
	Register(PathErrorCode, pathError{
		err: errors.New("some error"),
	})
}

type path struct {
	slashPath string
}

var _ id.Path = path{}

// OSPath implements id.Path.
func (p path) OSPath() string {
	return filepath.FromSlash(p.slashPath)
}

// SlashPath implements id.Path.
func (p path) SlashPath() string {
	return p.slashPath
}

// pathError almost always results from an OS-level function involving the file system.
type pathError struct {
	err      error
	relPaths []id.Path
}

var _ PathError = &pathError{}

func formatPaths(err PathError) string {
	pathStrs := make([]string, len(err.RelativePaths()))
	for i, path := range err.RelativePaths() {
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

// Error implements Error.
func (p pathError) Error() string {
	return Format(p, "%[1]s\nPotential causes:", p.err.Error())
}

// Code implements Error.
func (p pathError) Code() string {
	return PathErrorCode
}

// RelativePaths implements PathError.
func (p pathError) RelativePaths() []id.Path {
	return p.relPaths
}

// PathWrapf returns a PathError wrapping an error one or more relative paths.
func PathWrapf(err error, slashPaths ...string) PathError {
	if err == nil {
		return nil
	}
	paths := make([]id.Path, len(slashPaths))
	for i, p := range slashPaths {
		paths[i] = path{slashPath: p}
	}
	return pathError{err: err, relPaths: paths}
}
