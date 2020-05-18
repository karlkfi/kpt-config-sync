package filesystem_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	ft "github.com/google/nomos/pkg/importer/filesystem/filesystemtest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
	u, ok := o.(*unstructured.Unstructured)
	if ok {
		_ = unstructured.SetNestedStringMap(u.Object, map[string]string{
			"pods": "10",
		}, "spec", "hard")
		return
	}

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
		testFiles          ft.FileContentMap
		expectObject       *ast.FileObject
		expectedErrorCodes []string
	}{
		{
			testName: "Defining invalid yaml is an error.",
			testFiles: ft.FileContentMap{
				"namespaces/invalid.yaml": "This is not valid yaml.",
			},
			expectedErrorCodes: []string{status.PathErrorCode},
		},
		{
			testName: "Namespace dir with YAML Namespace",
			testFiles: ft.FileContentMap{
				"namespaces/bar/namespace.yaml": aNamespace("bar"),
			},
			expectObject: pointer(fake.NamespaceUnstructured("namespaces/bar")),
		},
		{
			testName: "Namespace dir with JSON Namespace",
			testFiles: ft.FileContentMap{
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
			expectObject: pointer(fake.NamespaceUnstructuredAtPath("namespaces/bar/namespace.json")),
		},
		{
			testName: "Namespaces dir with ignored file",
			testFiles: ft.FileContentMap{
				"namespaces/ignore": "",
			},
		},
		{
			testName: "Namespace dir with 2 ignored files",
			testFiles: ft.FileContentMap{
				"namespaces/bar/namespace.yaml": aNamespace("bar"),
				"namespaces/bar/ignore":         "",
				"namespaces/bar/ignore2":        "blah blah blah",
			},
			expectObject: pointer(fake.NamespaceUnstructured("namespaces/bar")),
		},
		{
			testName: "Namespace with labels/annotations",
			testFiles: ft.FileContentMap{
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
			expectObject: pointer(fake.NamespaceUnstructured("namespaces/bar", core.Label("env", "prod"), core.Annotation("audit", "true"))),
		},
		{
			testName: "Abstract Namespace dir with ignored file",
			testFiles: ft.FileContentMap{
				"namespaces/bar/ignore": "",
			},
		},
		{
			testName: "Namespaces dir with single ResourceQuota single file",
			testFiles: ft.FileContentMap{
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
			expectObject: pointer(ast.NewFileObject(fake.ResourceQuotaObjectUnstructured(specHardPods, core.Name("pod-quota")), cmpath.RelativeSlash("namespaces/bar/rq.yaml"))),
		},
		{
			testName: "Namespace dir with Custom Resource",
			testFiles: ft.FileContentMap{
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
			}, cmpath.RelativeSlash("namespaces/bar/philo.yaml"))),
		},
		{
			testName: "HierarchyConfig with multiple Kinds",
			testFiles: ft.FileContentMap{
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
			testFiles: ft.FileContentMap{
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
			testFiles: ft.FileContentMap{
				"namespaces/backend/namespace.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  annotations:
    number: "0000"
`,
			},
			expectObject: pointer(fake.NamespaceUnstructured("namespaces/backend", core.Annotation("number", "0000"))),
		},
		{
			testName: "metadata.annotations with boolean value",
			testFiles: ft.FileContentMap{
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
			testFiles: ft.FileContentMap{
				"namespaces/backend/namespace.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  annotations:
    boolean: "true"
`,
			},
			expectObject: pointer(fake.NamespaceUnstructured("namespaces/backend", core.Annotation("boolean", "true"))),
		},
		{
			testName: "metadata.labels with boolean value",
			testFiles: ft.FileContentMap{
				"namespaces/backend/ns.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  labels:
    boolean: true
`,
			},
			expectedErrorCodes: []string{filesystem.InvalidAnnotationValueErrorCode},
		},
		{
			testName: "metadata.labels with quoted boolean value",
			testFiles: ft.FileContentMap{
				"namespaces/backend/namespace.yaml": `
kind: Namespace
apiVersion: v1
metadata:
  name: backend
  labels:
    boolean: "true"
`,
			},
			expectObject: pointer(fake.NamespaceUnstructured("namespaces/backend", core.Label("boolean", "true"))),
		},
		{
			testName: "parses nested List",
			testFiles: ft.FileContentMap{
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
			expectObject: pointer(fake.NamespaceUnstructuredAtPath("namespaces/foo/list.yaml")),
		},
		{
			testName: "parses specialized List",
			testFiles: ft.FileContentMap{
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
			expectObject: pointer(fake.RoleUnstructuredAtPath("namespaces/foo/list.yaml", core.Name("my-role"))),
		},
		{
			testName: "illegal field in list-embedded resource",
			testFiles: ft.FileContentMap{
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
			d := ft.NewTestDir(t, ft.DirContents(tc.testFiles))

			var files []cmpath.Absolute
			for f := range tc.testFiles {
				files = append(files, d.Root().Join(cmpath.RelativeSlash(f)))
			}

			r := &filesystem.FileReader{}
			actual, mErr := r.Read(d.Root(), files)

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
