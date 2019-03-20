package status

import (
	"strings"
)

// PathErrorCode is the error code for a generic PathError.
const PathErrorCode = "2001"

func init() {
	Register(PathErrorCode, pathError{})
}

// pathError almost always results from an OS-level function involving the file system.
type pathError struct {
	err      error
	relPaths []string
}

var _ PathError = &pathError{}

// Error implements Error.
func (p pathError) Error() string {
	return Format(p, "%[1]s\nPotential causes:\n%[2]s", p.err.Error(), strings.Join(p.relPaths, "\n"))
}

// Code implements Error.
func (p pathError) Code() string {
	return PathErrorCode
}

// RelativePaths implements PathError.
func (p pathError) RelativePaths() []string {
	return p.relPaths
}

// PathWrapf returns a PathError wrapping an error one or more relative paths.
func PathWrapf(err error, relPaths ...string) PathError {
	if err == nil {
		return nil
	}
	return pathError{err: err, relPaths: relPaths}
}
