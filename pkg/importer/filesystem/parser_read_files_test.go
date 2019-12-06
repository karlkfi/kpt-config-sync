package filesystem_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/restmapper"
)

// Tests that don't make sense without literally writing to a hard disk.
// Or, ones that (for now) would require their own CL just to refactor to not require writing to a
// hard disk.

var engineerResource = &restmapper.APIGroupResources{
	Group: metav1.APIGroup{
		Name: "employees",
		Versions: []metav1.GroupVersionForDiscovery{
			{Version: "v1alpha1"},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1alpha1"},
	},
	VersionedResources: map[string][]metav1.APIResource{
		"v1alpha1": {
			{Name: "engineers", Namespaced: true, Kind: "Engineer"},
		},
	},
}

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

// Because Go makes taking the reference of the output of a function difficult.
func pointer(o ast.FileObject) *ast.FileObject {
	return &o
}

var specHardPods core.MetaMutator = func(o core.Object) {
	rq, ok := o.(*corev1.ResourceQuota)
	if !ok {
		panic(fmt.Sprintf("expected ResourceQuota, got %+v", o))
	}
	if rq.Spec.Hard == nil {
		rq.Spec.Hard = map[corev1.ResourceName]resource.Quantity{}
	}
	rq.Spec.Hard["pods"] = resource.MustParse("10")
}

// TestFilesystemReader tests reading from the file system.
func TestFilesystemReader(t *testing.T) {
	tests := []struct {
		testName           string
		testFiles          fstesting.FileContentMap
		expectObject       *ast.FileObject
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
				"namespaces/bar/namespace.yaml": aNamespace("bar"),
			},
			expectObject: pointer(fake.Namespace("namespaces/bar")),
		},
		{
			testName: "Namespace dir with JSON Namespace",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/namespace.json": `
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "bar"
  }
}
`,
			},
			expectObject: pointer(fake.NamespaceAtPath("namespaces/bar/namespace.json")),
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
				"namespaces/bar/namespace.yaml": aNamespace("bar"),
				"namespaces/bar/ignore":         "",
				"namespaces/bar/ignore2":        "blah blah blah",
			},
			expectObject: pointer(fake.Namespace("namespaces/bar")),
		},
		{
			testName: "Namespace with labels/annotations",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/namespace.yaml": `
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
			expectObject: pointer(fake.Namespace("namespaces/bar", core.Label("env", "prod"), core.Annotation("audit", "true"))),
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
				"namespaces/bar/rq.yaml": `
kind: ResourceQuota
apiVersion: v1
metadata:
  name: pod-quota
spec:
  hard:
    pods: "10"
`,
			},
			expectObject: pointer(ast.NewFileObject(fake.ResourceQuotaObject(specHardPods, core.Name("pod-quota")), cmpath.FromSlash("namespaces/bar/rq.yaml"))),
		},
		{
			testName: "Namespace dir with Custom Resource",
			testFiles: fstesting.FileContentMap{
				"namespaces/bar/philo.yaml": `
apiVersion: employees/v1alpha1
kind: Engineer
metadata:
  name: philo
spec:
  cafePreference: 3
`,
			},
			expectObject: pointer(ast.NewFileObject(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "employees/v1alpha1",
					"kind":       "Engineer",
					"metadata": map[string]interface{}{
						"name": "philo",
					},
					"spec": map[string]interface{}{
						"cafePreference": int64(3),
					},
				},
			}, cmpath.FromSlash("namespaces/bar/philo.yaml"))),
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
			expectObject: pointer(fake.HierarchyConfigAtPath("system/config.yaml", core.Name("config"),
				fake.HierarchyConfigResource(v1.HierarchyModeDefault, kinds.Role().GroupVersion(), kinds.Role().Kind, kinds.RoleBinding().Kind))),
		},
		{
			testName: "metadata.annotations with number value",
			testFiles: fstesting.FileContentMap{
				"namespaces/backend/ns.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  annotations:
    number: 0000
`,
			},
			expectedErrorCodes: []string{filesystem.InvalidAnnotationValueErrorCode},
		},
		{
			testName: "metadata.annotations with quoted number value",
			testFiles: fstesting.FileContentMap{
				"namespaces/backend/namespace.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  annotations:
    number: "0000"
`,
			},
			expectObject: pointer(fake.Namespace("namespaces/backend", core.Annotation("number", "0000"))),
		},
		{
			testName: "metadata.annotations with boolean value",
			testFiles: fstesting.FileContentMap{
				"namespaces/backend/ns.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  annotations:
    boolean: true
`,
			},
			expectedErrorCodes: []string{filesystem.InvalidAnnotationValueErrorCode},
		},
		{
			testName: "metadata.annotations with quoted boolean value",
			testFiles: fstesting.FileContentMap{
				"namespaces/backend/namespace.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  annotations:
    boolean: "true"
`,
			},
			expectObject: pointer(fake.Namespace("namespaces/backend", core.Annotation("boolean", "true"))),
		},
		{
			testName: "parses nested List",
			testFiles: fstesting.FileContentMap{
				"namespaces/foo/list.yaml": `
kind: List
apiVersion: v1
items:
- apiVersion: v1
  kind: List
  items:
  - apiVersion: v1
    kind: Namespace
    metadata:
      name: foo
`,
			},
			expectObject: pointer(fake.NamespaceAtPath("namespaces/foo/list.yaml")),
		},
		{
			testName: "parses specialized List",
			testFiles: fstesting.FileContentMap{
				"namespaces/foo/list.yaml": `
kind: RoleList
apiVersion: rbac.authorization.k8s.io/v1
items:
- apiVersion: rbac.authorization.k8s.io/v1
  kind: Role
  metadata:
    name: my-role
`,
			},
			expectObject: pointer(fake.RoleAtPath("namespaces/foo/list.yaml", core.Name("my-role"))),
		},
		{
			testName: "illegal field in list-embedded resource",
			testFiles: fstesting.FileContentMap{
				"namespaces/foo/list.yaml": `
kind: NamespaceList
apiVersion: v1
items:
- kind: Namespace
  apiVersion: v1
  metadata:
    name: foo
  status:
    phase: active
`,
			},
			expectedErrorCodes: []string{syntax.IllegalFieldsInConfigErrorCode},
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

			f := fstesting.NewTestClientGetter(t, engineerResource)
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
			actual, mErr := r.Read(rootPath.Join(cmpath.FromSlash(".")), false, nil)

			vettesting.ExpectErrors(tc.expectedErrorCodes, mErr, t)

			if tc.expectObject == nil {
				if len(actual) > 0 {
					t.Fatal("unexpected object")
				}
				return
			}

			if len(actual) == 0 {
				t.Fatal("expected object")
			}
			if diff := cmp.Diff(*tc.expectObject, actual[0], cmpopts.EquateEmpty(), resourcequota.ResourceQuantityEqual()); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
