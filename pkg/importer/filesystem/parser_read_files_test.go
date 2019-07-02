package filesystem_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/testoutput"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Tests that don't make sense without literally writing to a hard disk.
// Or, ones that (for now) would require their own CL just to refactor to not require writing to a
// hard disk.

type testDir struct {
	rootDir string
}

func (d testDir) remove(t *testing.T) {
	err := os.RemoveAll(d.rootDir)
	if err != nil {
		t.Error(err)
	}
}

func (d testDir) createTestFile(path, contents string, t *testing.T) {
	path = filepath.Join(d.rootDir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("error creating test dir %s: %v", path, err)
	}
	if err := ioutil.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("error creating test file %s: %v", path, err)
	}
}

func newTestDir(t *testing.T) *testDir {
	root, err := ioutil.TempDir("", "test_dir")
	if err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	return &testDir{root}
}

func aNamespace(name string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, name)
}

func TestEmptyDirectories(t *testing.T) {
	// Parsing should not encounter errors on seeing empty directories. If an error should occur, it
	// should be later.
	d := newTestDir(t)
	defer d.remove(t)

	for _, path := range []string{
		filepath.Join(d.rootDir, repo.SystemDir),
		filepath.Join(d.rootDir, repo.ClusterDir),
		filepath.Join(d.rootDir, repo.ClusterRegistryDir),
		filepath.Join(d.rootDir, repo.NamespacesDir),
	} {
		t.Run(path, func(t *testing.T) {
			if err := os.MkdirAll(path, 0750); err != nil {
				t.Fatalf("error creating test dir %s: %v", path, err)
			}

			f := fstesting.NewTestClientGetter(t)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			var err error
			rootPath, err := cmpath.NewRoot(cmpath.FromOS(d.rootDir))
			if err != nil {
				t.Error(err)
			}

			p := filesystem.NewParser(
				f,
				filesystem.ParserOpt{
					Vet:       false,
					Validate:  true,
					Extension: &filesystem.NomosVisitorProvider{},
					RootPath:  rootPath,
				},
			)

			if p.Errors() != nil {
				t.Fatalf("unexpected error: %v", p.Errors())
			}
		})
	}
}

// TestParserPerClusterAddressingVet tests nomos vet validation errors.
func TestFailOnInvalidYAML(t *testing.T) {
	tests := []struct {
		testName                 string
		testFiles                fstesting.FileContentMap
		expectedNamespaceConfigs map[string]v1.NamespaceConfig
		expectedSyncs            map[string]v1.Sync
		expectedErrorCodes       []string
	}{
		{
			testName: "Defining invalid yaml is an error.",
			testFiles: fstesting.FileContentMap{
				"namespaces/invalid.yaml": "This is not valid yaml.",
			},
			expectedErrorCodes: []string{status.APIServerErrorCode},
		},
		{
			testName: "No name is an error",
			testFiles: fstesting.FileContentMap{
				"cluster/clusterrole.yaml": `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
`,
			},
			expectedErrorCodes: []string{vet.MissingObjectNameErrorCode},
		},
		{
			testName: "Namespace dir with YAML Namespace",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ns.yaml": aNamespace("bar"),
			},
			expectedNamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/bar", nil)),
		},
		{
			testName: "Namespace dir with JSON Namespace",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ns.json": `
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "bar"
  }
}
`,
			},
			expectedNamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/bar", nil)),
		},
		{
			testName: "Namespaces dir with ignored file",
			testFiles: fstesting.FileContentMap{
				"namespaces/ignore": "",
			},
		},
		{
			testName: "Namespace dir with 2 ignored files",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ns.yaml": aNamespace("bar"),
				"namespaces/bar/ignore":  "",
				"namespaces/bar/ignore2": "blah blah blah",
			},
			expectedNamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/bar", nil)),
		},
		{
			testName: "Namespace dir with Namespace with labels/annotations",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ns.yaml": `
apiVersion: v1
kind: Namespace
metadata:
  name: bar
  labels:
    env: prod
  annotations:
    audit: "true"
`,
			},
			expectedNamespaceConfigs: testoutput.NamespaceConfigs(
				testoutput.NamespaceConfig("", "namespaces/bar",
					object.Mutations(object.Label("env", "prod"), object.Annotation("audit", "true"))),
			),
		},
		{
			testName: "custom resource w/o a CRD applied",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/undefined.yaml": `
kind: Undefined
apiVersion: non.existent
metadata:
  name: undefined
`,
				"namespaces/bar/ns.yaml": aNamespace("bar"),
			},
			expectedErrorCodes: []string{status.APIServerErrorCode},
		},
		{
			testName: "Abstract Namespace dir with ignored file",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ignore": "",
			},
		},
		{
			testName: "Namespaces dir with single ResourceQuota single file",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/combo.yaml": aNamespace("bar") + "\n---\n" + `
kind: ResourceQuota
apiVersion: v1
metadata:
  name: pod-quota
spec:
  hard:
    pods: "10"
`,
			},
			expectedNamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/bar", nil,
				resourceQuotaObject(object.Name("pod-quota"), testoutput.Source("namespaces/bar/combo.yaml")))),
			expectedSyncs: testoutput.Syncs(kinds.ResourceQuota()),
		},
		{
			testName: "Namespace dir with Custom Resource",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ns.yaml": aNamespace("bar"),
				"namespaces/bar/philo.yaml": `
apiVersion: employees/v1alpha1
kind: Engineer
metadata:
  name: philo
spec:
  cafePreference: 3
`,
			},
			expectedNamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/bar", nil,
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "employees/v1alpha1",
						"kind":       "Engineer",
						"metadata": map[string]interface{}{
							"annotations": map[string]interface{}{"configmanagement.gke.io/source-path": "namespaces/bar/philo.yaml"},
							"name":        "philo",
						},
						"spec": map[string]interface{}{
							"cafePreference": int64(3),
						}}},
			)),
			expectedSyncs: testoutput.Syncs(schema.GroupVersionKind{
				Group:   "employees",
				Version: "v1alpha1",
				Kind:    "Engineer",
			}),
		},
		{
			testName: "HierarchyConfig with multiple Kinds",
			testFiles: fstesting.FileContentMap{
				"system/config.yaml": `
kind: HierarchyConfig
apiVersion: configmanagement.gke.io/v1
metadata:
  name: config
spec:
  resources:
  - group: rbac.authorization.k8s.io
    kinds: [ "Role", "RoleBinding" ]
`,
			},
		},
	}
	for _, tc := range tests {
		tc.testFiles["system/repo.yaml"] = `
kind: Repo
apiVersion: configmanagement.gke.io/v1
spec:
  version: "1.0.0"
metadata:
  name: repo
`
		t.Run(tc.testName, func(t *testing.T) {

			d := newTestDir(t)
			defer d.remove(t)

			if glog.V(6) {
				glog.Infof("Testcase: %+v", spew.Sdump(tc))
			}

			for k, v := range tc.testFiles {
				d.createTestFile(k, v, t)
			}

			f := fstesting.NewTestClientGetter(t)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			var err error
			rootPath, err := cmpath.NewRoot(cmpath.FromOS(d.rootDir))
			if err != nil {
				t.Error(err)
			}

			p := filesystem.NewParser(
				f,
				filesystem.ParserOpt{
					Vet:       true,
					Validate:  true,
					Extension: &filesystem.NomosVisitorProvider{},
					RootPath:  rootPath,
				},
			)
			actualConfigs, mErr := p.Parse("", &namespaceconfig.AllConfigs{}, time.Time{}, "")

			vettesting.ExpectErrors(tc.expectedErrorCodes, mErr, t)
			if mErr != nil || tc.expectedErrorCodes != nil {
				// We expected there to be an error, so no need to do config validation
				return
			}

			if tc.expectedNamespaceConfigs == nil {
				tc.expectedNamespaceConfigs = testoutput.NamespaceConfigs()
			}
			if tc.expectedSyncs == nil {
				tc.expectedSyncs = testoutput.Syncs()
			}

			expectedConfigs := &namespaceconfig.AllConfigs{
				NamespaceConfigs: tc.expectedNamespaceConfigs,
				ClusterConfig:    testoutput.ClusterConfig(),
				CRDClusterConfig: testoutput.CRDClusterConfig(),
				Syncs:            tc.expectedSyncs,
				Repo:             fake.RepoObject(),
			}
			if diff := cmp.Diff(expectedConfigs, actualConfigs, resourcequota.ResourceQuantityEqual(), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Actual and expected configs didn't match: diff\n%v", diff)
			}
		})
	}
}
