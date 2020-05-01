package testing

import (
	"os"
	"path"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// FakeReader is a fake implementation of filesystem.Reader.
type FakeReader map[cmpath.Path][]ast.FileObject

var _ filesystem.Reader = &FakeReader{}

// NewFakeReader initializes a FakeReader from a set of FileObjects.
func NewFakeReader(root cmpath.Path, objs []ast.FileObject) FakeReader {
	result := make(FakeReader)
	for _, obj := range objs {
		p := cmpath.FromSlash(path.Join(root.SlashPath(), obj.SlashPath()))
		result[p] = append(result[p], obj)
	}
	return result
}

func (r FakeReader) Read(_ cmpath.Path, paths []cmpath.Path) ([]ast.FileObject, status.MultiError) {
	var result []ast.FileObject
	for _, p := range paths {
		if objs, ok := r[p]; ok {
			result = append(result, objs...)
		} else {
			return nil, status.PathWrapError(os.ErrNotExist, p.OSPath())
		}
	}
	return result, nil
}

// ToFileList returns the list of files available to the FakeReader.
func (r FakeReader) ToFileList() []cmpath.Path {
	var result []cmpath.Path
	for p := range r {
		result = append(result, p)
	}
	return result
}
