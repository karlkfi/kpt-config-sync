package status

import (
	"path/filepath"

	"github.com/google/nomos/pkg/importer/id"
)

// PathErrorCode is the error code for a generic PathError.
const PathErrorCode = "2001"

var pathError = NewErrorBuilder(PathErrorCode)

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

// PathWrapError returns a PathError wrapping an error one or more relative paths.
func PathWrapError(err error, slashPaths ...string) Error {
	if err == nil {
		return nil
	}
	paths := make([]id.Path, len(slashPaths))
	for i, p := range slashPaths {
		paths[i] = path{slashPath: p}
	}
	return pathError.Wrap(err).BuildWithPaths(paths...)
}
