package filesystem_test

import (
	"os"
	"path"
	"testing"

	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	ft "github.com/google/nomos/pkg/importer/filesystem/filesystemtest"
)

func TestFileReader_Read_NotExist(t *testing.T) {
	dir := ft.NewTestDir(t).Root()

	reader := filesystem.FileReader{}

	objs, err := reader.Read(dir, []cmpath.Absolute{dir.Join(cmpath.RelativeSlash("no-exist"))})
	if err != nil || len(objs) > 0 {
		t.Errorf("got Read(nonexistent path) = %+v, %v; want nil, nil", objs, err)
	}
}

func TestFileReader_Read_BadPermissionsParent(t *testing.T) {
	// If we're root, this test will fail, because we'll have read access anyway.
	if os.Geteuid() == 0 {
		t.Skipf("%s will fail running with EUID==0", t.Name())
	}

	tmpRelative := "tmp.yaml"

	dir := ft.NewTestDir(t,
		ft.FileContents(tmpRelative, ""),
		ft.Chmod(tmpRelative, 0000),
	).Root()

	reader := filesystem.FileReader{}

	objs, err := reader.Read(dir, []cmpath.Absolute{
		dir.Join(cmpath.RelativeSlash(tmpRelative)),
	})
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on parent dir) = %+v, %v; want nil, error", objs, err)
	}
}

func TestFileReader_Read_BadPermissionsChild(t *testing.T) {
	// If we're root, this test will fail, because we'll have read access anyway.
	if os.Geteuid() == 0 {
		t.Skipf("%s will fail running with EUID==0", t.Name())
	}

	subDir := "namespaces"
	tmpRelative := path.Join(subDir, "tmp.yaml")

	dir := ft.NewTestDir(t,
		ft.FileContents(tmpRelative, ""),
		ft.Chmod(subDir, 0000),
	).Root()

	reader := filesystem.FileReader{}

	objs, err := reader.Read(dir, []cmpath.Absolute{
		dir.Join(cmpath.RelativeSlash(tmpRelative)),
	})
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on child dir) = %+v, %v; want nil, error", objs, err)
	}
}
