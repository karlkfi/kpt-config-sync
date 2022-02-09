// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmpath

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/importer/id"
	"github.com/pkg/errors"
)

// Absolute represents an absolute path on a file system.
type Absolute struct {
	// path is a slash-delimited path.
	path string
}

var _ id.Path = Absolute{}

// AbsoluteSlash returns an Absolute path from a slash-delimited path.
//
// It is an error to pass a non-absolute path.
func AbsoluteSlash(p string) (Absolute, error) {
	if !filepath.IsAbs(p) {
		return Absolute{}, errors.Errorf("not an absolute path")
	}
	return Absolute{path: path.Clean(p)}, nil
}

// AbsoluteOS returns an Absolute path from an OS-specific path.
//
// Converts p to an absolute path if it is not already.
// Assumes the current working directory is the path p is relative to.
func AbsoluteOS(p string) (Absolute, error) {
	return AbsoluteSlash(filepath.ToSlash(p))
}

// OSPath implements id.Path.
func (p Absolute) OSPath() string {
	return filepath.FromSlash(p.path)
}

// SlashPath implements id.Path.
func (p Absolute) SlashPath() string {
	return p.path
}

// Join appends r to p, creating a new Absolute path.
func (p Absolute) Join(r Relative) Absolute {
	return Absolute{path: path.Join(p.path, r.path)}
}

// Split returns a slice of the path elements.
func (p Absolute) Split() []string {
	splits := strings.Split(p.path, "/")
	if splits[len(splits)-1] == "" {
		// Discard trailing empty string if this is a path ending in slash.
		splits = splits[:len(splits)-1]
	}
	return splits
}

// Equal returns true if the underlying absolute paths are equal.
func (p Absolute) Equal(other Absolute) bool {
	// Assumes Path was constructed or altered via exported methods.
	return p.path == other.path
}

// EvalSymlinks evaluates any symlinks in the Absolute and returns a new
// Absolute with no symlinks.
func (p Absolute) EvalSymlinks() (Absolute, error) {
	abs, err := filepath.EvalSymlinks(filepath.FromSlash(p.path))
	if err != nil {
		return Absolute{}, err
	}
	return AbsoluteOS(abs)
}
