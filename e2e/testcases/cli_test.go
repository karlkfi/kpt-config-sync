package e2e

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	nomostesting "github.com/google/nomos/e2e/nomostest/testing"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
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
	// 3) nomos vet --no-api-server-check
	// 4) nomos hydrate --no-api-server-check
	// 5) nomos vet --no-api-server-check --path=<hydrated-dir>
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
		flags.OutputYAML,
		[]*unstructured.Unstructured{
			fake.UnstructuredObject(kinds.Namespace(), core.Name("foo")),
			fake.UnstructuredObject(kinds.ConfigMap(), core.Name("cm1"), core.Namespace("foo")),
		})
	if err != nil {
		tw.Fatal(err)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", fmt.Sprintf("--path=%s", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "hydrate", "--no-api-server-check",
		fmt.Sprintf("--path=%s", tmpDir), fmt.Sprintf("--output=%s/compiled", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "hydrate", "--no-api-server-check", "--flat",
		fmt.Sprintf("--path=%s", tmpDir), fmt.Sprintf("--output=%s/compiled.yaml", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("cat", fmt.Sprintf("%s/compiled.yaml", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	expectedYaml := []byte(`---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
  namespace: foo
---
apiVersion: v1
kind: Namespace
metadata:
  name: foo
`)

	if diff := cmp.Diff(string(expectedYaml), string(out)); diff != "" {
		tw.Errorf("nomos hydrate diff: %s", diff)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", "--source-format=unstructured",
		fmt.Sprintf("--path=%s/compiled", tmpDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}
}

func TestNomosHydrateWithClusterSelectors(t *testing.T) {
	tmpDir := nomostest.TestDir(t)
	tw := nomostesting.New(t)

	_ = nomostest.NewOptStruct(nomostest.TestClusterName(tw), tmpDir, tw)

	configPath := "../../examples/hierarchical-repo-with-cluster-selectors"
	expectedCompiledDir := "../../examples/hierarchical-repo-with-cluster-selectors-compiled"
	compiledDir := fmt.Sprintf("%s/compiled", tmpDir)
	clusterDevCompiledDir := fmt.Sprintf("%s/cluster-dev", compiledDir)
	clusterStagingCompiledDir := fmt.Sprintf("%s/cluster-staging", compiledDir)
	clusterProdCompiledDir := fmt.Sprintf("%s/cluster-prod", compiledDir)

	compiledWithAPIServerCheckDir := fmt.Sprintf("%s/compiled-with-api-server-check", tmpDir)

	compiledDirWithoutClustersFlag := fmt.Sprintf("%s/compiled-without-clusters-flag", tmpDir)
	expectedCompiledWithoutClustersFlagDir := "../../examples/hierarchical-repo-with-cluster-selectors-compiled-without-clusters-flag"

	compiledJSONDir := fmt.Sprintf("%s/compiled-json", tmpDir)
	compiledJSONWithoutClustersFlagDir := fmt.Sprintf("%s/compiled-json-without-clusters-flag", tmpDir)
	expectedCompiledJSONDir := "../../examples/hierarchical-repo-with-cluster-selectors-compiled-json"
	expectedCompiledWithoutClustersFlagJSONDir := "../../examples/hierarchical-repo-with-cluster-selectors-compiled-json-without-clusters-flag"

	// Test `nomos vet --no-api-server-check`
	out, err := exec.Command("nomos", "vet", "--no-api-server-check", fmt.Sprintf("--path=%s", configPath)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos vet --no-api-server-check --clusters=cluster-dev`
	out, err = exec.Command("nomos", "vet", "--no-api-server-check", fmt.Sprintf("--path=%s", configPath), "--clusters=cluster-dev").CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos vet`
	out, err = exec.Command("nomos", "vet", fmt.Sprintf("--path=%s", configPath)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos hydrate --no-api-server-check --clusters=cluster-dev,cluster-prod,cluster-staging`
	out, err = exec.Command("nomos", "hydrate", "--no-api-server-check",
		fmt.Sprintf("--path=%s", configPath),
		"--clusters=cluster-dev,cluster-prod,cluster-staging",
		fmt.Sprintf("--output=%s", compiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", compiledDir, expectedCompiledDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos hydrate --clusters=cluster-dev,cluster-prod,cluster-staging`
	out, err = exec.Command("nomos", "hydrate",
		fmt.Sprintf("--path=%s", "../../examples/hierarchical-repo-with-cluster-selectors"),
		"--clusters=cluster-dev,cluster-prod,cluster-staging",
		fmt.Sprintf("--output=%s", compiledWithAPIServerCheckDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", compiledWithAPIServerCheckDir, expectedCompiledDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos hydrate`
	out, err = exec.Command("nomos", "hydrate",
		fmt.Sprintf("--path=%s", "../../examples/hierarchical-repo-with-cluster-selectors"),
		fmt.Sprintf("--output=%s", compiledDirWithoutClustersFlag)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", compiledDirWithoutClustersFlag, expectedCompiledWithoutClustersFlagDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", fmt.Sprintf("%s/cluster-dev", compiledDirWithoutClustersFlag), fmt.Sprintf("%s/cluster-dev", compiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", fmt.Sprintf("%s/cluster-staging", compiledDirWithoutClustersFlag), fmt.Sprintf("%s/cluster-staging", compiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", fmt.Sprintf("%s/cluster-prod", compiledDirWithoutClustersFlag), fmt.Sprintf("%s/cluster-prod", compiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos hydrate --format=json --clusters=cluster-dev,cluster-prod,cluster-staging`
	out, err = exec.Command("nomos", "hydrate", "--format=json",
		fmt.Sprintf("--path=%s", "../../examples/hierarchical-repo-with-cluster-selectors"),
		"--clusters=cluster-dev,cluster-prod,cluster-staging",
		fmt.Sprintf("--output=%s", compiledJSONDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", compiledJSONDir, expectedCompiledJSONDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos hydrate --format=json`
	out, err = exec.Command("nomos", "hydrate", "--format=json",
		fmt.Sprintf("--path=%s", "../../examples/hierarchical-repo-with-cluster-selectors"),
		fmt.Sprintf("--output=%s", compiledJSONWithoutClustersFlagDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", compiledJSONWithoutClustersFlagDir, expectedCompiledWithoutClustersFlagJSONDir).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos vet --no-api-server-check --source-format=unstructured` on the hydrated configs
	out, err = exec.Command("nomos", "vet", "--no-api-server-check", "--source-format=unstructured", fmt.Sprintf("--path=%s", clusterDevCompiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", "--source-format=unstructured", fmt.Sprintf("--path=%s", clusterStagingCompiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--no-api-server-check", "--source-format=unstructured", fmt.Sprintf("--path=%s", clusterProdCompiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos vet --source-format=unstructured` on the hydrated configs
	out, err = exec.Command("nomos", "vet", "--source-format=unstructured", fmt.Sprintf("--path=%s", clusterDevCompiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--source-format=unstructured", fmt.Sprintf("--path=%s", clusterStagingCompiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("nomos", "vet", "--source-format=unstructured", fmt.Sprintf("--path=%s", clusterProdCompiledDir)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}
}

func testSyncFromNomosHydrateOutput(t *testing.T, config string) {
	nt := nomostest.New(t, ntopts.Unstructured)

	if err := nt.ValidateNotFound("bookstore1", "", &corev1.Namespace{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.ValidateNotFound("bookstore2", "", &corev1.Namespace{}); err != nil {
		nt.T.Fatal(err)
	}

	nt.Root.Copy(config, "acme")
	nt.Root.CommitAndPush("Add cluster-dev configs")
	nt.WaitForRepoSyncs()

	if err := nt.Validate("bookstore1", "", &corev1.Namespace{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.Validate("bookstore2", "", &corev1.Namespace{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.ValidateNotFound("quota", "bookstore1", &corev1.ResourceQuota{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.Validate("quota", "bookstore2", &corev1.ResourceQuota{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.Validate("cm-all", "bookstore1", &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.Validate("cm-dev-staging", "bookstore1", &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.ValidateNotFound("cm-prod", "bookstore1", &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.ValidateNotFound("cm-dev", "bookstore1", &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.ValidateNotFound("cm-disabled", "bookstore1", &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}

	if err := nt.ValidateNotFound("cm-all", "bookstore2", &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}
}

func TestSyncFromNomosHydrateOutputYAMLDir(t *testing.T) {
	testSyncFromNomosHydrateOutput(t, "../../examples/hierarchical-repo-with-cluster-selectors-compiled/cluster-dev/.")
}

func TestSyncFromNomosHydrateOutputJSONDir(t *testing.T) {
	testSyncFromNomosHydrateOutput(t, "../../examples/hierarchical-repo-with-cluster-selectors-compiled-json/cluster-dev/.")
}

func testSyncFromNomosHydrateOutputFlat(t *testing.T, format string) {
	tmpDir := nomostest.TestDir(t)
	tw := nomostesting.New(t)

	configPath := "../../examples/hierarchical-repo-with-cluster-selectors"
	compiledConfigFile := fmt.Sprintf("%s/compiled.%s", tmpDir, format)

	out, err := exec.Command("nomos", "hydrate", "--no-api-server-check", "--flat",
		fmt.Sprintf("--path=%s", configPath),
		fmt.Sprintf("--format=%s", format),
		"--clusters=cluster-dev",
		fmt.Sprintf("--output=%s", compiledConfigFile)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	testSyncFromNomosHydrateOutput(t, compiledConfigFile)
}

func TestSyncFromNomosHydrateOutputJSONFlat(t *testing.T) {
	testSyncFromNomosHydrateOutputFlat(t, "json")
}

func TestSyncFromNomosHydrateOutputYAMLFlat(t *testing.T) {
	testSyncFromNomosHydrateOutputFlat(t, "yaml")
}

func TestNomosHydrateWithUnknownScopedObject(t *testing.T) {
	tmpDir := nomostest.TestDir(t)
	tw := nomostesting.New(t)

	_ = nomostest.NewOptStruct(nomostest.TestClusterName(tw), tmpDir, tw)

	compiledDirWithoutAPIServerCheck := fmt.Sprintf("%s/compiled-without-api-server-check", tmpDir)
	compiledDirWithAPIServerCheck := fmt.Sprintf("%s/compiled-with-api-server-check", tmpDir)

	kubevirtPath := "../../examples/kubevirt"

	// Test `nomos vet --no-api-server-check`
	out, err := exec.Command("nomos", "vet", "--no-api-server-check", fmt.Sprintf("--path=%s", kubevirtPath)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Verify that `nomos vet` returns a KNV1021 error.
	out, err = exec.Command("nomos", "vet", fmt.Sprintf("--path=%s", kubevirtPath)).CombinedOutput()
	if err == nil {
		tw.Error(fmt.Errorf("`nomos vet --path=%s` expects an error, got nil", kubevirtPath))
	} else {
		if !strings.Contains(string(out), "Error: 1 error(s)") || !strings.Contains(string(out), "KNV1021") {
			tw.Error(fmt.Errorf("`nomos vet --path=%s` expects only one KNV1021 error, got %v", kubevirtPath, string(out)))
		}
	}

	// Verify that `nomos hydrate --no-api-server-check` generates no error, and the output dir includes all the objects no matter their scopes.
	out, err = exec.Command("nomos", "hydrate", "--no-api-server-check",
		fmt.Sprintf("--path=%s", "../../examples/kubevirt"),
		fmt.Sprintf("--output=%s", compiledDirWithoutAPIServerCheck)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	out, err = exec.Command("diff", "-r", compiledDirWithoutAPIServerCheck, "../../examples/kubevirt-compiled").CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Verify that `nomos hydrate` generates a KNV1021 error, and the output dir includes all the objects no matter their scopes.
	out, err = exec.Command("nomos", "hydrate",
		fmt.Sprintf("--path=%s", "../../examples/kubevirt"),
		fmt.Sprintf("--output=%s", compiledDirWithAPIServerCheck)).CombinedOutput()
	if err == nil {
		tw.Error(fmt.Errorf("`nomo hydrate --path=%s` expects an error, got nil", kubevirtPath))
	} else {
		if !strings.Contains(string(out), ": 1 error(s)") || !strings.Contains(string(out), "KNV1021") {
			tw.Error(fmt.Errorf("`nomos hydrate --path=%s` expects only one KNV1021 error, got %v", kubevirtPath, string(out)))
		}
	}

	out, err = exec.Command("diff", "-r", compiledDirWithAPIServerCheck, "../../examples/kubevirt-compiled").CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Test `nomos vet --no-api-server-check` on the hydrated configs.
	out, err = exec.Command("nomos", "vet", "--no-api-server-check", "--source-format=unstructured", fmt.Sprintf("--path=%s", compiledDirWithoutAPIServerCheck)).CombinedOutput()
	if err != nil {
		tw.Log(string(out))
		tw.Error(err)
	}

	// Verify that `nomos vet` on the hydrated configs returns a KNV1021 error.
	out, err = exec.Command("nomos", "vet", "--source-format=unstructured", fmt.Sprintf("--path=%s", compiledDirWithoutAPIServerCheck)).CombinedOutput()
	if err == nil {
		tw.Error(fmt.Errorf("`nomos vet --path=%s` expects an error, got nil", compiledDirWithoutAPIServerCheck))
	} else {
		if !strings.Contains(string(out), "Error: 1 error(s)") || !strings.Contains(string(out), "KNV1021") {
			tw.Error(fmt.Errorf("`nomos vet --path=%s` expects only one KNV1021 error, got %v", compiledDirWithoutAPIServerCheck, string(out)))
		}
	}
}
