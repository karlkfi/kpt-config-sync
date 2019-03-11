package status

// PathErrorCode is the error code for a generic PathError.
const PathErrorCode = "2001"

type pathError struct {
	err      error
	relPaths []string
}

var _ PathError = &pathError{}

func (p pathError) Error() string {
	return Format(p, "error: %s", p.err.Error())
}

func (p pathError) Code() string {
	return PathErrorCode
}

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
