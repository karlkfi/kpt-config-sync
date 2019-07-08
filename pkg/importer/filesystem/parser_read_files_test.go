package filesystem_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
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

// TestFilesystemReader tests reading from the file system.
func TestFilesystemReader(t *testing.T) {
	tests := []struct {
		testName           string
		testFiles          fstesting.FileContentMap
		expectedErrorCodes []string
	}{
		{
			testName: "Defining invalid yaml is an error.",
			testFiles: fstesting.FileContentMap{
				"namespaces/invalid.yaml": "This is not valid yaml.",
			},
			expectedErrorCodes: []string{status.APIServerErrorCode},
		},
		{
			testName: "Namespace dir with YAML Namespace",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/ns.yaml": aNamespace("bar"),
			},
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

			r := &filesystem.FileReader{
				ClientGetter: f,
			}
			_, mErr := r.Read(rootPath.Join(cmpath.FromSlash(".")), false)

			vettesting.ExpectErrors(tc.expectedErrorCodes, mErr, t)
		})
	}
}
