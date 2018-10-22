package testing

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// FileContentMap specifies files that should be created as part of a parser
// test.
type FileContentMap map[string]string

// TestDir creates a new test directory for putting test files in using
// `CreateTestFile()`.
type TestDir struct {
	tmpDir  string
	rootDir string
	*testing.T
}

// NewTestDir constructs a new test directory in the passed root directory.
func NewTestDir(t *testing.T, root string) *TestDir {
	tmp, err := ioutil.TempDir("", "test_dir")
	if err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	root = filepath.Join(tmp, root)
	if err = os.Mkdir(root, 0750); err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	tree := filepath.Join(root, "tree")
	if err = os.Mkdir(tree, 0750); err != nil {
		t.Fatalf("Failed to create tree dir %v", err)
	}
	return &TestDir{tmp, root, t}
}

// Remove deletes the test directory.
func (d TestDir) Remove() {
	if err := os.RemoveAll(d.tmpDir); err != nil {
		d.Logf("Unable to remove %v; may require manual deletion: %v", d.Root(), err)
	}
}

// CreateTestFile creates a file as the path with the contents written.
func (d TestDir) CreateTestFile(path, contents string) {
	path = filepath.Join(d.rootDir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		d.Fatalf("error creating test dir %s: %v", path, err)
	}
	if err := ioutil.WriteFile(path, []byte(contents), 0644); err != nil {
		d.Fatalf("error creating test file %s: %v", path, err)
	}
}

// Root returns the full path to the test directory.
func (d TestDir) Root() string {
	return d.rootDir
}
