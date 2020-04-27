package filesystem

import (
	"io/ioutil"
	"os"
	"path"
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

func asRooted(t *testing.T, path string) cmpath.RootedPath {
	root, err := cmpath.NewRoot(cmpath.FromSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestFileReader_Read_NotExist(t *testing.T) {
	dir := tempDir(t)

	// Remove directory so os.Stat returns a NotExist error.
	err := os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}

	reader := FileReader{}

	objs, err := reader.Read(asRooted(t, dir))
	if err != nil || len(objs) > 0 {
		t.Errorf("got Read(nonexistent path) = %+v, %v; want nil, nil", objs, err)
	}
}

func TestFileReader_Read_BadPermissionsParent(t *testing.T) {
	dir := tempDir(t)

	// If we're root, this test will fail, because we'll have read access anyway.
	if os.Geteuid() == 0 {
		t.Skip("Read_BadPermissionsParent will fail running with EUID==0")
	}

	// Change permissions on the root directory so os.Stat returns a
	// Permission error.
	err := os.Chmod(dir, 000)
	if err != nil {
		t.Fatal(err)
	}

	reader := FileReader{}

	objs, err := reader.Read(asRooted(t, dir))
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on parent dir) = %+v, %v; want nil, error", objs, err)
	}
}

func TestFileReader_Read_BadPermissionsChild(t *testing.T) {
	dir := tempDir(t)

	// If we're root, this test will fail, because we'll have read access anyway.
	if os.Geteuid() == 0 {
		t.Skip("Read_BadPermissionsChild will fail running with EUID==0")
	}

	// Create subdirectory.
	subDir := path.Join(dir, "namespaces")
	err := os.Mkdir(subDir, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	// Change permissions on the subdirectory so os.Stat returns a
	// Permission error.
	err = os.Chmod(subDir, 000)
	if err != nil {
		t.Fatal(err)
	}

	reader := FileReader{}

	objs, err := reader.Read(asRooted(t, dir))
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on child dir) = %+v, %v; want nil, error", objs, err)
	}
}
