package status

// PathErrorCode is the error code for a generic PathError.
const PathErrorCode = "2001"

// pathError almost always results from an OS-level function involving the file system.
type pathError struct {
	err      error
	relPaths []string
}

var _ PathError = &pathError{}

// Error implements Error.
func (p pathError) Error() string {
	return Format(p, p.err.Error())
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
