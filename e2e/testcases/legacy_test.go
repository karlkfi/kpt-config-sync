package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
)

type BatsTest struct {
	fileName string
}

func (bt *BatsTest) Run(t *testing.T) {
	t.Parallel()

	nt := nomostest.New(t)
	nomosDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal("Failed to get nomos dir: ", err)
	}
	cmd := exec.Command(filepath.Join(nomosDir, filepath.FromSlash("third_party/bats-core/bin/bats")), "--tap", bt.fileName)

	// Factored out for accessing deprecated functions that only exist for supporting bats tests.
	privateKeyPath := nt.GitPrivateKeyPath() //nolint:staticcheck
	gitRepoPort := nt.GitRepoPort()          //nolint:staticcheck
	kubeConfigPath := nt.KubeconfigPath()    //nolint:staticcheck

	// TODO: create pipes for stdout / stderr / tap output and redirect lines through t.Log
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set fd3 (tap output) to stdout
	cmd.ExtraFiles = []*os.File{os.Stderr}
	cmd.Env = []string{
		// For now omit test case filtering, but it would look something like the following were we to add it.
		//fmt.Sprintf("E2E_TEST_FILTER=%s", testCaseRegex),
		fmt.Sprintf("BATS_TMPDIR=%s", filepath.Join(nt.TmpDir, "bats")),
		"TIMING=true",
		// tell git to use the ssh private key and not check host key
		fmt.Sprintf("GIT_SSH_COMMAND=ssh -q -o StrictHostKeyChecking=no -i %s", privateKeyPath),
		// passes the path to e2e manifests to the bats tests
		fmt.Sprintf("MANIFEST_DIR=%s", filepath.Join(nomosDir, filepath.FromSlash("e2e/raw-nomos/manifests"))),
		// passes the git server SSH port to bash tests
		fmt.Sprintf("FWD_SSH_PORT=%d", gitRepoPort),
		// for running 'nomos' command from built binary
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		// provide kubeconfig path to kubectl
		fmt.Sprintf("KUBECONFIG=%s", kubeConfigPath),
		// kubectl creates the .kube directory in HOME if it does not exist
		fmt.Sprintf("HOME=%s", filepath.Join(nt.TmpDir, "fake-home")),
	}

	t.Log("Using environment")
	for _, env := range cmd.Env {
		t.Logf("  %s", env)
	}

	t.Logf("Starting legacy test %s", bt.fileName)
	err = cmd.Run()
	if err != nil {
		t.Fatal(err, cmd)
	}
}

func TestBats(t *testing.T) {
	t.Parallel()
	testCases := []*BatsTest{
		{fileName: "acme.bats"},
		//{fileName: "apiservice.bats"},
		//{fileName: "basic.bats"},
		//{fileName: "cli.bats"},
		//{fileName: "cluster_resources.bats"},
		//{fileName: "custom_resource_definitions_v1.bats"},
		//{fileName: "custom_resource_definitions_v1beta1.bats"},
		//{fileName: "custom_resources_v1.bats"},
		//{fileName: "custom_resources_v1beta1.bats"},
		//{fileName: "foo_corp.bats"},
		//{fileName: "gatekeeper.bats"},
		//{fileName: "multiversion.bats"},
		//{fileName: "namespaces.bats"},
		//{fileName: "operator-no-policy-dir.bats"},
		//{fileName: "per_cluster_addressing.bats"},
		//{fileName: "preserve_fields.bats"},
		//{fileName: "repoless.bats"},
		//{fileName: "resource_conditions.bats"},
		//{fileName: "schema_validation.bats"},
		//{fileName: "status_monitoring.bats"},
	}
	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.fileName, tc.Run)
	}
}
