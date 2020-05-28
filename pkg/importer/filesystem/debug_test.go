package filesystem_test

import (
	"testing"

	"github.com/google/nomos/pkg/importer/filesystem"
	ft "github.com/google/nomos/pkg/importer/filesystem/filesystemtest"
)

func TestWalkDirectory(t *testing.T) {
	// add .git/ and .git/test_dir to /tmp/nomos-test-XXXX directory for testing.
	dir := ft.NewTestDir(t,
		ft.FileContents(".git/test_dir.yaml", "test content"),
	).Root()

	d, err := filesystem.WalkDirectory(dir.OSPath())
	if err != nil {
		t.Fatalf("got WalkDirectory() = %v, want nil", err)
	}
	// Check whether /tmp/nomos-test-XXXX/test_dir.yaml skipped.
	if len(d) > 2 {
		t.Errorf("got WalkDirectory(.git sub-directories traversed) = %v, want 2", d)
	}
}
