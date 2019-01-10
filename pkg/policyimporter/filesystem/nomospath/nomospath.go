package nomospath

import (
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

// Sourced represents an object associated with a path in a Nomos repository.
type Sourced interface {
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
func (p Root) Rel(targpath string) (Relative, error) {
	relpath, err := filepath.Rel(p.path, targpath)
	if err != nil {
		return Relative{}, errors.Wrapf(err, "unable to get relative path in repo")
	}
	return Relative{path: relpath, root: p}, nil
}

// NewRoot creates a new Root.
// path is either the path to Nomos relative to system root or the path relative to the working
//   directory.
// Returns error if path is not absolute and the program is unable to retrieve the working directory.
func NewRoot(path string) (Root, error) {
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
func makeCleanAbsolute(path string) (string, error) {
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		// Recall that filepath.Join cleans the resulting path.
		return filepath.Join(wd, path), nil
	}
	return filepath.Clean(path), nil
}

// Relative is a path relative to a Root.
type Relative struct {
	// The OS-specific path relative to the Nomos repository root.
	path string

	// The underlying Nomos repository this path is relative to.
	root Root
}

// NewFakeRelative returns a fake Relative which is not actually relative to
// a real Nomos root. For testing and documentation.
// path MUST be OS-independent.
func NewFakeRelative(path string) Relative {
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

// ToRelativeSlashPaths returns the slash paths relative to Nomos root for every passed
// Relative. This is a temporary method with the sole purpose of allowing refactoring to be
// broken into reasonably small pieces.
func ToRelativeSlashPaths(relpaths []Relative) []string {
	result := make([]string, len(relpaths))
	for i, relpath := range relpaths {
		result[i] = relpath.RelativeSlashPath()
	}
	return result
}

// Equal returns true if the underlying relative path and root directories are identical.
func (p Relative) Equal(that Relative) bool {
	return p.path == that.path && cmp.Equal(p.root, that.root)
}
