package testing

import (
	"os"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
)

// FakeReader is a fake implementation of filesystem.Reader.
type FakeReader map[cmpath.Absolute][]ast.FileObject

var _ filesystem.Reader = &FakeReader{}

// NewFakeReader initializes a FakeReader from a set of FileObjects.
func NewFakeReader(root cmpath.Absolute, objs []ast.FileObject) FakeReader {
	result := make(FakeReader)
	for _, obj := range objs {
		p := root.Join(obj.Relative)
		result[p] = append(result[p], obj)
	}
	return result
}

func (r FakeReader) Read(_ cmpath.Absolute, paths []cmpath.Absolute) ([]ast.FileObject, status.MultiError) {
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
func (r FakeReader) ToFileList() []cmpath.Absolute {
	var result []cmpath.Absolute
	for p := range r {
		result = append(result, p)
	}
	return result
}
