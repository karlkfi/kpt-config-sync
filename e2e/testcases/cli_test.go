package e2e

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/google/nomos/cmd/nomos/hydrate"
	"github.com/google/nomos/e2e/nomostest"
	nomostesting "github.com/google/nomos/e2e/nomostest/testing"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNomosInitVet(t *testing.T) {
	// Ensure that the following sequence of commands succeeds:
	//
	// 1) git init
	// 2) nomos init
	// 3) nomos vet --no-api-server-check
	tmpDir := nomostest.TestDir(t)
	tw := nomostesting.New(t)

	out, err := exec.Command("git", "init", tmpDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "init", fmt.Sprintf("--path=%s", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", fmt.Sprintf("--path=%s", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}
}

func TestNomosInitHydrate(t *testing.T) {
	// Ensure that the following sequence of commands succeeds:
	//
	// 1) git init
	// 2) nomos init
	// 3) nomos hydrate --no-api-server-check
	tmpDir := nomostest.TestDir(t)
	tw := nomostesting.New(t)

	out, err := exec.Command("git", "init", tmpDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "init", fmt.Sprintf("--path=%s", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	err = hydrate.PrintFile(fmt.Sprintf("%s/namespaces/foo/ns.yaml", tmpDir),
		[]*unstructured.Unstructured{
			fake.UnstructuredObject(kinds.Namespace(), core.Name("foo")),
		})
	if err != nil {
		tw.Fatal(err)
	}

	out, err = exec.Command("nomos", "hydrate", "--no-api-server-check",
		fmt.Sprintf("--path=%s", tmpDir), fmt.Sprintf("--output=%s/compiled", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", "--source-format=unstructured",
		fmt.Sprintf("--path=%s/compiled", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}
}
