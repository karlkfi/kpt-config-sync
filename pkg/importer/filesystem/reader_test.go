package filesystem_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/google/nomos/pkg/importer/filesystem"
	ft "github.com/google/nomos/pkg/importer/filesystem/filesystemtest"
	"github.com/google/nomos/pkg/status"
)

func TestFileReader_Read_NotExist(t *testing.T) {
	dir := ft.NewTestDir(t)
	fps := dir.FilePaths("no-exist")

	reader := filesystem.FileReader{}

	objs, err := reader.Read(fps)
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
	)
	fps := dir.FilePaths(tmpRelative)

	reader := filesystem.FileReader{}

	objs, err := reader.Read(fps)
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
	)
	fps := dir.FilePaths(tmpRelative)

	reader := filesystem.FileReader{}

	objs, err := reader.Read(fps)
	if err == nil || len(objs) > 0 {
		t.Errorf("got Read(bad permissions on child dir) = %+v, %v; want nil, error", objs, err)
	}
}

func TestFileReader_Read_ValidMetadata(t *testing.T) {
	testCases := []struct {
		name     string
		metadata string
	}{
		{
			name: "no labels/annotations",
		},
		{
			name:     "empty labels",
			metadata: "labels:",
		},
		{
			name:     "empty annotations",
			metadata: "annotations:",
		},
		{
			name:     "empty map labels",
			metadata: "labels: {}",
		},
		{
			name:     "empty map annotations",
			metadata: "annotations: {}",
		},
	}

	nsFile := "namespace.yaml"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := ft.NewTestDir(t,
				ft.FileContents(nsFile, fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: foo
  %s
`, tc.metadata)))
			fps := dir.FilePaths(nsFile)
			reader := filesystem.FileReader{}
			_, err := reader.Read(fps)

			if err != nil {
				t.Fatalf("got Read() = %v, want nil", err)
			}
		})
	}
}

func TestFileReader_Read_InvalidAnnotations(t *testing.T) {
	nsFile := "namespace.yaml"
	dir := ft.NewTestDir(t,
		ft.FileContents(nsFile, `
apiVersion: v1
kind: Namespace
metadata:
  name: foo
  annotations: a
`))
	fps := dir.FilePaths(nsFile)
	reader := filesystem.FileReader{}
	_, err := reader.Read(fps)

	if err == nil {
		t.Fatal("got Read() = nil, want err")
	}
	errs := err.Errors()
	if len(errs) != 1 {
		t.Fatalf("got Read() = %d errors, want 1 err", len(errs))
	}

	if _, isResourceError := errs[0].(status.ResourceError); !isResourceError {
		t.Fatalf("got Read() = %T, want ResourceError", errs[0])
	}
}

func TestFileReader_Read_InvalidObject(t *testing.T) {
	nsFile := "namespace.yaml"
	dir := ft.NewTestDir(t,
		ft.FileContents(nsFile, `
apiVersion: configmanagement.gke.io/v1
kind: Repo
metadata:
  name: repo
spec:
  version: 1.0
`))
	fps := dir.FilePaths(nsFile)
	reader := filesystem.FileReader{}
	_, err := reader.Read(fps)

	if err == nil {
		t.Fatal("got Read() = nil, want err")
	}
	errs := err.Errors()
	if len(errs) != 1 {
		t.Fatalf("got Read() = %d errors, want 1 err", len(errs))
	}

	if _, isResourceError := errs[0].(status.ResourceError); !isResourceError {
		t.Fatalf("got Read() = %T, want ResourceError", errs[0])
	}
}

func TestFileReader_Read_ValidObject(t *testing.T) {
	nsFile := "namespace.yaml"
	dir := ft.NewTestDir(t,
		ft.FileContents(nsFile, `
apiVersion: configmanagement.gke.io/v1
kind: Namespace
metadata:
  name: ns
spec:
  version: "1.0"
`))
	fps := dir.FilePaths(nsFile)
	reader := filesystem.FileReader{}
	_, err := reader.Read(fps)

	if err != nil {
		t.Fatalf("got Read() = %v, want nil", err)
	}
}
