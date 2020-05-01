package cmpath

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/importer/id"
)

// Path is a path in a filesystem.
type Path struct {
	// path is a slash-delimited path.
	path string
}

var _ id.Path = Path{}

// FromSlash returns a Path from a slash-delimited path.
func FromSlash(p string) Path {
	var fp string
	if p != "" {
		fp = path.Clean(p)
	}
	return Path{path: fp}
}

// FromOS constructs a Path from an OS-dependent path.
func FromOS(p string) Path {
	return Path{path: filepath.ToSlash(filepath.Clean(p))}
}

// Join appends elem to the Path.
func (p Path) Join(elem string) Path {
	return Path{path: path.Join(p.path, elem)}
}

// SlashPath returns an os-independent representation of this Path.
func (p Path) SlashPath() string {
	return p.path
}

// OSPath returns an os-specific representation of this Path.
func (p Path) OSPath() string {
	return filepath.FromSlash(p.path)
}

// Dir returns the directory containing this Path.
func (p Path) Dir() Path {
	return Path{path: filepath.Dir(p.path)}
}

// Base returns the Base of this Path.
func (p Path) Base() string {
	return filepath.Base(p.path)
}

// IsRoot returns true if the path is the Nomos root directory.
func (p Path) IsRoot() bool {
	return p.path == "."
}

// Split returns a slice of the path elements.
func (p Path) Split() []string {
	splits := strings.Split(p.path, "/")
	if splits[len(splits)-1] == "" {
		// Discard trailing empty string if this is a path ending in slash.
		splits = splits[:len(splits)-1]
	}
	return splits
}

// Equal returns true if the underlying paths are equal.
func (p Path) Equal(other Path) bool {
	// Assumes Path was constructed or altered via exported methods.
	return p.path == other.path
}

// Abs converts a cmpath.Path of a directory to the absolute path, after
// following symlinks.
func Abs(p Path) (Path, error) {
	// Evaluate any symlinks in the directory.
	relDir, err := filepath.EvalSymlinks(p.OSPath())
	if err != nil {
		return FromOS(""), err
	}
	// Ensure we're working with the absolute path, as most symlinks are relative
	// paths and filepath.EvalSymlinks does not get the absolute destination.
	absDir, err := filepath.Abs(relDir)
	if err != nil {
		return FromOS(""), err
	}
	return FromOS(absDir), nil
}
