package path

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Sourced represents an object associated with a path in a Nomos repository.
type Sourced interface {
	RelativeSlashPath() string
}

// NomosRoot is a path to a directory holding a Nomos repository.
// Robust to changes in the working directory.
type NomosRoot struct {
	// The underlying absolute OS-specific path to the Nomos repository.
	path string
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p NomosRoot) AbsoluteOSPath() string {
	return p.path
}

// Join joins a path element to the existing NomosRoot, returning a NomosRelative.
func (p NomosRoot) Join(elem string) NomosRelative {
	return NomosRelative{path: filepath.Clean(elem), root: p}
}

// Rel breaks the passed target path into a NomosRelative
func (p NomosRoot) Rel(targpath string) (NomosRelative, error) {
	path, err := filepath.Rel(p.path, targpath)
	if err != nil {
		return NomosRelative{}, errors.Wrapf(err, "unable to get relative path in repo")
	}
	return NomosRelative{path: path, root: p}, nil
}

// NewNomosRoot creates a new NomosRoot.
// path is either the path to Nomos relative to system root or the path relative to the working
//   directory.
// Returns error if path is not absolute and the program is unable to retrieve the working directory.
func NewNomosRoot(path string) (NomosRoot, error) {
	absolutePath, err := makeCleanAbsolute(path)
	if err != nil {
		return NomosRoot{}, err
	}
	return NomosRoot{path: absolutePath}, nil
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

// NomosRelative is a path relative to a NomosRoot.
type NomosRelative struct {
	// The OS-specific path relative to the Nomos repository root.
	path string

	// The underlying Nomos repository this path is relative to.
	root NomosRoot
}

// NewFakeNomosRelativePath returns a fake NomosRelative which is not actually relative to
// a real Nomos root. For testing and documentation.
func NewFakeNomosRelativePath(path string) NomosRelative {
	return NomosRelative{path: filepath.Clean(path), root: NomosRoot{}}
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p NomosRelative) AbsoluteOSPath() string {
	return filepath.Join(p.root.path, p.path)
}

// RelativeSlashPath returns the OS-independent path relative to the Nomos root.
func (p NomosRelative) RelativeSlashPath() string {
	return filepath.ToSlash(p.path)
}

// ToRelativeSlashPaths returns the slash paths relative to Nomos root for every passed
// NomosRelative. This is a temporary method with the sole purpose of allowing refactoring to be
// broken into reasonably small pieces.
func ToRelativeSlashPaths(paths []NomosRelative) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = path.RelativeSlashPath()
	}
	return result
}
