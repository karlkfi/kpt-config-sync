package e2e

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
)

type BatsTest struct {
	fileName string
	nomosDir string
}

func (bt *BatsTest) batsPath() string {
	return filepath.Join(bt.nomosDir, filepath.FromSlash("third_party/bats-core/bin/bats"))
}

func (bt *BatsTest) Run(t *testing.T) {
	t.Parallel()

	countCmd := exec.Command(bt.batsPath(), "--count", bt.fileName)
	out, err := countCmd.CombinedOutput()
	if err != nil {
		t.Errorf("%v: %s", countCmd, string(out))
		t.Fatal("Failed to get test count from bats:", err)
	}
	testCount, err := strconv.Atoi(strings.Trim(string(out), "\n"))
	if err != nil {
		t.Fatalf("Failed to parse test count %q from bats: %v", out, err)
	}
	t.Logf("Found %d testcases in %s", testCount, bt.fileName)
	for testNum := 1; testNum <= testCount; testNum++ {
		t.Run(strconv.Itoa(testNum), bt.runTest(testNum))
	}
}

func (bt *BatsTest) runTest(testNum int) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		nt := nomostest.New(t)

		// Factored out for accessing deprecated functions that only exist for supporting bats tests.
		privateKeyPath := nt.GitPrivateKeyPath() //nolint:staticcheck
		gitRepoPort := nt.GitRepoPort()          //nolint:staticcheck
		kubeConfigPath := nt.KubeconfigPath()    //nolint:staticcheck

		batsTmpDir := filepath.Join(nt.TmpDir, "bats", "tmp")
		batsHome := filepath.Join(nt.TmpDir, "bats", "home")
		for _, dir := range []string{batsTmpDir, batsHome} {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				t.Fatalf("failed to create dir %s for bats testing: %v", dir, err)
			}
		}

		pipeRead, pipeWrite, err := os.Pipe()
		if err != nil {
			t.Fatal("failed to create pipe for test output", err)
		}
		defer func() {
			if pipeWrite != nil {
				_ = pipeWrite.Close()
			}
			_ = pipeRead.Close()
		}()
		cmd := exec.Command(bt.batsPath(), "--tap", bt.fileName)
		cmd.Stdout = pipeWrite
		cmd.Stderr = pipeWrite
		// Set fd3 (tap output) to stdout
		cmd.ExtraFiles = []*os.File{pipeWrite}
		cmd.Env = []string{
			// Indicate the exact test num to run.
			fmt.Sprintf("E2E_RUN_TEST_NUM=%d", testNum),
			// instruct bats to use the per-testcase temp directory rather than /tmp
			fmt.Sprintf("TMPDIR=%s", batsTmpDir),
			// instruct our e2e tests to report timing information
			"TIMING=true",
			// tell git to use the ssh private key and not check host key
			fmt.Sprintf("GIT_SSH_COMMAND=ssh -q -o StrictHostKeyChecking=no -i %s", privateKeyPath),
			// passes the path to e2e manifests to the bats tests
			fmt.Sprintf("MANIFEST_DIR=%s", filepath.Join(bt.nomosDir, filepath.FromSlash("e2e/raw-nomos/manifests"))),
			// passes the git server SSH port to bash tests
			fmt.Sprintf("FWD_SSH_PORT=%d", gitRepoPort),
			// for running 'nomos' command from built binary
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			// provide kubeconfig path to kubectl
			fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath),
			// kubectl creates the .kube directory in HOME if it does not exist
			fmt.Sprintf("HOME=%s", batsHome),
		}

		t.Log("Using environment")
		for _, env := range cmd.Env {
			t.Logf("  %s", env)
		}

		t.Logf("Starting legacy test %s", bt.fileName)
		err = cmd.Start()
		if err != nil {
			t.Fatalf("failed to start command %s: %v", cmd, err)
		}

		// Close our copy of pipe so our read end of the pipe will get EOF when the subprocess terminates (and closes
		// it's copy of the write end).  Bats will still have the write end of the pipe hooked up to stdout/stderr/fd3
		// until it exits.
		_ = pipeWrite.Close()
		pipeWrite = nil

		reader := bufio.NewReader(pipeRead)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					if len(line) != 0 {
						t.Log(string(line))
					}
					break
				}
				t.Fatal("error reading from bats subprocess:", err)
			}
			t.Log(strings.TrimRight(string(line), "\n"))
		}

		err = cmd.Wait()
		if err != nil {
			t.Fatalf("command failed %s: %v", cmd, err)
		}
	}
}

func TestBats(t *testing.T) {
	t.Parallel()

	nomosDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal("Failed to get nomos dir: ", err)
	}

	testCases := []*BatsTest{
		{fileName: "acme.bats"},
		{fileName: "apiservice.bats"},
		{fileName: "basic.bats"},
		{fileName: "cli.bats"},
		{fileName: "cluster_resources.bats"},
		{fileName: "custom_resource_definitions_v1.bats"},
		{fileName: "custom_resource_definitions_v1beta1.bats"},
		{fileName: "custom_resources_v1.bats"},
		{fileName: "custom_resources_v1beta1.bats"},
		{fileName: "foo_corp.bats"},
		{fileName: "gatekeeper.bats"},
		{fileName: "multiversion.bats"},
		{fileName: "namespaces.bats"},
		{fileName: "operator-no-policy-dir.bats"},
		{fileName: "per_cluster_addressing.bats"},
		{fileName: "preserve_fields.bats"},
		{fileName: "repoless.bats"},
		{fileName: "resource_conditions.bats"},
		{fileName: "status_monitoring.bats"},
	}
	for idx := range testCases {
		tc := testCases[idx]
		tc.nomosDir = nomosDir
		t.Run(tc.fileName, tc.Run)
	}
}
