package nomospath

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/status"
)

// Sourced represents an object associated with a path in a Nomos repository.
type Sourced interface {
	// RelativeSlashPath returns the slash-delimited path relative to Nomos root.
	RelativeSlashPath() string
}

// Root is a path to a directory holding a Nomos repository.
// Robust to changes in the working directory.
type Root struct {
	// The underlying absolute OS-specific path to the Nomos repository.
	path string
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p Root) AbsoluteOSPath() string {
	return p.path
}

// Join joins a path element to the existing Root, returning a Relative.
func (p Root) Join(elem string) Relative {
	return Relative{path: filepath.Clean(elem), root: p}
}

// Rel breaks the passed target path into a Relative
func (p Root) Rel(targPath string) (Relative, status.PathError) {
	relPath, err := filepath.Rel(p.path, targPath)
	if err != nil {
		return Relative{}, status.PathWrapf(err, p.path, targPath)
	}
	return Relative{path: relPath, root: p}, nil
}

// NewRoot creates a new Root.
// path is either the path to Nomos relative to system root or the path relative to the working
//   directory.
// Returns error if path is not absolute and the program is unable to retrieve the working directory.
func NewRoot(path string) (Root, status.Error) {
	absolutePath, err := makeCleanAbsolute(path)
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
			return "", status.OSWrapf(err)
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
	// The OS-specific path relative to the Nomos repository root.
	path string

	// The underlying Nomos repository this path is relative to.
	root Root
}

// NewRelative returns a fake Relative which is not actually relative to
// a real Nomos root. For testing and documentation.
// path MUST be OS-independent.
func NewRelative(path string) Relative {
	return Relative{path: filepath.Clean(filepath.FromSlash(path))}
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p Relative) AbsoluteOSPath() string {
	return filepath.Join(p.root.path, p.path)
}

// RelativeSlashPath returns the OS-independent path relative to the Nomos root.
func (p Relative) RelativeSlashPath() string {
	return filepath.ToSlash(p.path)
}

// Dir returns the directory containing this Relative.
func (p Relative) Dir() Relative {
	return Relative{path: filepath.Dir(p.path), root: p.root}
}

// Base returns the Base path of this location.
func (p Relative) Base() string {
	return filepath.Base(p.path)
}

// IsRoot returns true if the path is the Nomos root directory.
func (p Relative) IsRoot() bool {
	return p.path == "."
}

// Split returns the path elements relative to Nomos root.
func (p Relative) Split() []string {
	return strings.Split(p.path, string(os.PathSeparator))
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
	return Relative{root: p.root, path: filepath.Join(p.path, elem)}
}
