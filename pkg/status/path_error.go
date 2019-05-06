package status

import (
	"path/filepath"
	"sort"
	"strings"

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

// PathWrapf returns a PathError wrapping an error one or more relative paths.
func PathWrapf(err error, slashPaths ...string) Error {
	if err == nil {
		return nil
	}
	paths := make([]id.Path, len(slashPaths))
	for i, p := range slashPaths {
		paths[i] = path{slashPath: p}
	}
	return pathError.WithPaths(paths...)(err)
}
