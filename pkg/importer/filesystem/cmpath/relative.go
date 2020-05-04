package cmpath

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/importer/id"
)

// Relative represents a relative path on a file system.
// The path is not guaranteed to be relative to the working directory.
type Relative struct {
	// path is a slash-delimited path.
	path string
}

var _ id.Path = Relative{}

// RelativeSlash returns an Relative path from a slash-delimited path.
func RelativeSlash(p string) Relative {
	return Relative{path: path.Clean(p)}
}

// RelativeOS returns an Relative path from an OS-specific path.
func RelativeOS(p string) Relative {
	return RelativeSlash(filepath.ToSlash(p))
}

// OSPath implements id.Path.
func (p Relative) OSPath() string {
	return filepath.FromSlash(p.path)
}

// SlashPath implements id.Path.
func (p Relative) SlashPath() string {
	return p.path
}

// Join appends r to p, creating a new Relative path.
func (p Relative) Join(r Relative) Relative {
	return Relative{path: path.Join(p.path, r.path)}
}

// Split returns a slice of the path elements.
func (p Relative) Split() []string {
	splits := strings.Split(p.path, "/")
	if splits[len(splits)-1] == "" {
		// Discard trailing empty string if this is a path ending in slash.
		splits = splits[:len(splits)-1]
	}
	return splits
}

// Equal returns true if the underlying relative paths are equal.
func (p Relative) Equal(other Relative) bool {
	// Assumes Path was constructed or altered via exported methods.
	return p.path == other.path
}

// Base returns the Base of this Path.
func (p Relative) Base() string {
	return path.Base(p.path)
}

// Dir returns the directory containing this Path.
func (p Relative) Dir() Relative {
	return RelativeSlash(path.Dir(p.path))
}

// IsRoot returns true if the path is the Nomos root directory.
func (p Relative) IsRoot() bool {
	return p.path == "."
}
