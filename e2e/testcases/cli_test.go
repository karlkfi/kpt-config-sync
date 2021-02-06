package e2e

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
)

func TestNomosInitVet(t *testing.T) {
	// Ensure that the following sequence of commands succeeds:
	//
	// 1) git init
	// 2) nomos init
	// 3) nomos vet
	tmpDir := nomostest.TestDir(t)

	out, err := exec.Command("git", "init", tmpDir).CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Error(err)
	}

	out, err = exec.Command("nomos", "init", fmt.Sprintf("--path=%s", tmpDir)).CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", fmt.Sprintf("--path=%s", tmpDir)).CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Error(err)
	}
}
