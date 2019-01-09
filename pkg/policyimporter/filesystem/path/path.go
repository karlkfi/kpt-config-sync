package path

import (
	"os"
	"path/filepath"
)

// Sourced represents an object associated with a path in a Nomos repository.
type Sourced interface {
	RelativeSlashPath() string
}

// NomosRootPath is a path to a directory holding a Nomos repository.
// Robust to changes in the working directory.
type NomosRootPath struct {
	// The underlying absolute OS-specific path to the Nomos repository.
	path string
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p NomosRootPath) AbsoluteOSPath() string {
	return p.path
}

// Join joins a path element to the existing NomosRootPath, returning a NomosRelativePath.
func (p NomosRootPath) Join(elem string) NomosRelativePath {
	return NomosRelativePath{path: filepath.Join(p.path, elem), root: p}
}

// NewNomosRootPath creates a new NomosRootPath.
// path is either the path to Nomos relative to system root or the path relative to the working
//   directory.
// Returns error if path is not absolute and the program is unable to retrieve the working directory.
func NewNomosRootPath(path string) (NomosRootPath, error) {
	absolutePath, err := makeCleanAbsolute(path)
	if err != nil {
		return NomosRootPath{}, err
	}
	return NomosRootPath{path: absolutePath}, nil
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

// NomosRelativePath is a path relative to a NomosRootPath.
type NomosRelativePath struct {
	// The OS-specific path relative to the Nomos repository root.
	path string

	// The underlying Nomos repository this path is relative to.
	root NomosRootPath
}

// NewFakeNomosRelativePath returns a fake NomosRelativePath which is not actually relative to
// a real Nomos root. For testing and documentation.
func NewFakeNomosRelativePath(path string) NomosRelativePath {
	return NomosRelativePath{path: filepath.Clean(path), root: NomosRootPath{}}
}

// AbsoluteOSPath returns the absolute OS-specific path.
func (p NomosRelativePath) AbsoluteOSPath() string {
	return filepath.Join(p.root.path, p.path)
}

// RelativeSlashPath returns the OS-independent path relative to the Nomos root.
func (p NomosRelativePath) RelativeSlashPath() string {
	return filepath.ToSlash(p.path)
}
