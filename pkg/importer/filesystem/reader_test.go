package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
)

var tmpBase = filepath.Join(os.TempDir(), "nomos-test")

func tempDir(t *testing.T) string {
	err := os.MkdirAll(tmpBase, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir(tmpBase, "")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFileReader_Read_NotExist(t *testing.T) {
	dir, err := cmpath.AbsoluteOS(tempDir(t))
	if err != nil {
		t.Fatal(err)
	}

	reader := FileReader{}

	objs, err := reader.Read(dir, []cmpath.Absolute{dir.Join(cmpath.RelativeSlash("no-exist"))})
	if err != nil || len(objs) > 0 {
		t.Errorf("got Read(nonexistent path) = %+v, %v; want nil, nil", objs, err)
	}
}

func TestFileReader_Read_BadPermissionsParent(t *testing.T) {
	dir, err := cmpath.AbsoluteOS(tempDir(t))
	if err != nil {
		t.Fatal(err)
	}

	// If we're root, this test will fail, because we'll have read access anyway.
	if os.Geteuid() == 0 {
		t.Skip("Read_BadPermissionsParent will fail running with EUID==0")
	}

	tmpFile := dir.Join(cmpath.RelativeSlash("tmp.yaml"))
	_, err = os.Create(tmpFile.OSPath())
	if err != nil {
		t.Fatal(err)
	}

	// Change permissions on the root directory so os.Stat returns a
	// Permission error.
	err = os.Chmod(dir.OSPath(), 000)
	if err != nil {
		t.Fatal(err)
	}

	reader := FileReader{}

	objs, err := reader.Read(dir, []cmpath.Absolute{tmpFile})
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on parent dir) = %+v, %v; want nil, error", objs, err)
	}
}

func TestFileReader_Read_BadPermissionsChild(t *testing.T) {
	dir, err := cmpath.AbsoluteOS(tempDir(t))
	if err != nil {
		t.Fatal(err)
	}

	// If we're root, this test will fail, because we'll have read access anyway.
	if os.Geteuid() == 0 {
		t.Skip("Read_BadPermissionsChild will fail running with EUID==0")
	}

	// Create subdirectory.
	subDir := dir.Join(cmpath.RelativeSlash("namespaces"))
	err = os.Mkdir(subDir.OSPath(), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile := dir.Join(cmpath.RelativeSlash("tmp.yaml"))
	err = ioutil.WriteFile(tmpFile.OSPath(), nil, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	// Change permissions on the subdirectory so os.Stat returns a
	// Permission error.
	err = os.Chmod(tmpFile.OSPath(), 000)
	if err != nil {
		t.Fatal(err)
	}

	reader := FileReader{}

	objs, err := reader.Read(dir, []cmpath.Absolute{tmpFile})
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on child dir) = %+v, %v; want nil, error", objs, err)
	}
}
