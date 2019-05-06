package cmpath

import (
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// Root is a path to a directory holding a Nomos repository.
// Robust to changes in the working directory.
type Root struct {
	// The underlying absolute OS-specific path to the Nomos repository.
	path string
}

// Join joins a path element to the existing Root, returning a Relative.
func (p Root) Join(rel Path) Relative {
	return Relative{path: rel, root: p}
}

// Rel breaks the passed target path into a Relative
func (p Root) Rel(targPath Path) (Relative, status.Error) {
	relPath, err := filepath.Rel(p.path, targPath.OSPath())
	if err != nil {
		return Relative{}, status.PathWrapf(err, p.path, targPath.SlashPath())
	}
	return Relative{path: FromOS(relPath), root: p}, nil
}

// NewRoot creates a new Root.
// path is either the path to Nomos relative to system root or the path relative to the working
//   directory.
// Returns error if path is not absolute and the program is unable to retrieve the working directory.
func NewRoot(path Path) (Root, status.Error) {
	absolutePath, err := makeCleanAbsolute(path.OSPath())
	if err != nil {
		return Root{}, err
	}
	return Root{path: absolutePath}, nil
}

// Equal returns true if the underlying paths are identical.
func (p Root) Equal(that Root) bool {
	return p.path == that.path
}

// makeCleanAbsolute returns the cleaned, absolute path.
func makeCleanAbsolute(path string) (string, status.Error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		// path is relative to home directory.
		// filepath.Abs does not cover this case.
		usr, err := user.Current()
		if err != nil {
			return "", status.OSWrap(err)
		}
		home := usr.HomeDir
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", status.PathWrapf(err, path)
	}
	return filepath.Clean(absPath), nil
}

// Relative is a path relative to a Root.
type Relative struct {
	// The path relative to the Nomos repository root.
	path Path

	// The underlying Nomos repository this path is relative to.
	root Root
}

// Path returns a copy of the underlying Path relative to the Nomos root.
func (p Relative) Path() Path {
	return p.path
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p Relative) AbsoluteOSPath() string {
	return filepath.Join(p.root.path, p.path.OSPath())
}

// Equal returns true if the underlying relative path and root directories are identical.
func (p Relative) Equal(that Relative) bool {
	return p.path == that.path && cmp.Equal(p.root, that.root)
}

// Root returns a copy of the underlying root path this Relative is based from.
func (p Relative) Root() Root {
	return Root{path: p.root.path}
}

// Join returns a copy of the underlying Relative with the additional path element appended.
func (p Relative) Join(elem string) Relative {
	return Relative{root: p.root, path: p.path.Join(elem)}
}

// Path is a path in a filesystem.
type Path struct {
	// path is a slash-delimited path.
	path string
}

var _ id.Path = Path{}

// FromSlash returns a Path from a slash-delimited path.
func FromSlash(p string) Path {
	return Path{path: path.Clean(p)}
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
	return strings.Split(p.path, "/")
}

// Equal returns true if the underlying paths are equal.
func (p Path) Equal(other Path) bool {
	// Assumes Path was constructed or altered via exported methods.
	return p.path == other.path
}
