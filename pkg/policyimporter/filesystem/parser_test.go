// Reviewed by sunilarora
/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filesystem

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/go-test/deep"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchyv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	fstesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	aNamespaceTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Name}}
`

	aNamespaceWithLabelsAndAnnotations = `
apiVersion: v1
kind: Namespace
metadata:
  name: bar
  labels:
    env: prod
  annotations:
    audit: "true"
`

	aNamespaceJSONTemplate = `
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "{{.Name}}"
  }
}
`
	aQuotaTemplate = `
kind: ResourceQuota
apiVersion: v1
metadata:
  name: pod-quota{{.ID}}
  namespace: {{.Namespace}}
spec:
  hard:
    pods: "10"
`

	aRoleTemplate = `
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creator{{.ID}}
  namespace: {{.Namespace}}
rules:
- apiGroups: ["batch/v1"]
  resources: ["jobs"]
  verbs:
   - "*"
`

	aRoleBindingTemplate = `
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creators{{.ID}}
  namespace: {{.Namespace}}
subjects:
- kind: Group
  name: bob@acme.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: job-creator
  apiGroup: rbac.authorization.k8s.io
`

	aClusterRoleTemplate = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creator{{.ID}}
rules:
- apiGroups: ["batch/v1"]
  resources: ["jobs"]
  verbs:
   - "*"
`
	aClusterRoleBindingTemplate = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creators{{.ID}}
  namespace: {{.Namespace}}
subjects:
- kind: Group
  name: bob@acme.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: job-creator
  apiGroup: rbac.authorization.k8s.io
`
	aPodSecurityPolicyTemplate = `
apiVersion: extensions/v1beta1
kind: PodSecurityPolicy
metadata:
  name: psp{{.ID}}
spec:
  privileged: false
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  runAsUser:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny
  volumes:
  - '*'
`

	aConfigMapTemplate = `
kind: ConfigMap
apiVersion: v1
data:
  {{.Namespace}}: {{.Attribute}}
metadata:
  name: {{.Name}}
`

	aNamespaceSelectorTemplate = `
kind: NamespaceSelector
apiVersion: nomos.dev/v1alpha1
metadata:
  name: sre-supported
spec:
  selector:
    matchLabels:
      environment: prod
`

	aNomosConfig = `
kind: NomosConfig
apiVersion: nomos.dev/v1alpha1
spec:
  repoVersion: "1.0.0"
`

	aSyncTemplate = `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: {{.Kind}}
spec:
  groups:
  - group: {{.Group}}
    kinds:
    - kind: {{.Kind}}
      versions:
      - version: {{.Version}}
`

	aDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: {{.Namespace}}
spec:
  replicas: 3
`

	aPhiloTemplate = `
apiVersion: employees/v1alpha1
kind: Engineer
metadata:
  name: philo
  namespace: {{.Namespace}}
spec:
  cafePreference: 3
`

	aNodeTemplate = `
apiVersion: v1
kind: Node
metadata:
  name: gke-1234
`
)

var (
	aNamespace          = template.Must(template.New("aNamespace").Parse(aNamespaceTemplate))
	aNamespaceJSON      = template.Must(template.New("aNamespaceJSON").Parse(aNamespaceJSONTemplate))
	aQuota              = template.Must(template.New("aQuota").Parse(aQuotaTemplate))
	aRole               = template.Must(template.New("aRole").Parse(aRoleTemplate))
	aRoleBinding        = template.Must(template.New("aRoleBinding").Parse(aRoleBindingTemplate))
	aClusterRole        = template.Must(template.New("aClusterRole").Parse(aClusterRoleTemplate))
	aClusterRoleBinding = template.Must(template.New("aClusterRoleBinding").Parse(aClusterRoleBindingTemplate))
	aPodSecurityPolicy  = template.Must(template.New("aPodSecurityPolicyTemplate").Parse(aPodSecurityPolicyTemplate))
	aConfigMap          = template.Must(template.New("aConfigMap").Parse(aConfigMapTemplate))
	aDeployment         = template.Must(template.New("aDeployment").Parse(aDeploymentTemplate))
	aSync               = template.Must(template.New("aSync").Parse(aSyncTemplate))
	aPhilo              = template.Must(template.New("aPhilo").Parse(aPhiloTemplate))
	aNode               = template.Must(template.New("aNode").Parse(aNodeTemplate))
)

type templateData struct {
	ID, Name, Namespace, Attribute, Group, Version, Kind string
}

func (d templateData) apply(t *template.Template) string {
	var b bytes.Buffer
	if err := t.Execute(&b, d); err != nil {
		panic(errors.Wrapf(err, "template data: %#v", d))
	}
	return b.String()
}

type testDir struct {
	tmpDir  string
	rootDir string
	*testing.T
}

func newTestDir(t *testing.T, root string) *testDir {
	tmp, err := ioutil.TempDir("", "test_dir")
	if err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	root = filepath.Join(tmp, root)
	if err = os.Mkdir(root, 0750); err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	tree := filepath.Join(root, "tree")
	if err = os.Mkdir(tree, 0750); err != nil {
		t.Fatalf("Failed to create tree dir %v", err)
	}
	return &testDir{tmp, root, t}
}

func (d testDir) remove() {
	os.RemoveAll(d.tmpDir)
}

func (d testDir) createTestFile(path, contents string) {
	path = filepath.Join(d.rootDir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		d.Fatalf("error creating test dir %s: %v", path, err)
	}
	if err := ioutil.WriteFile(path, []byte(contents), 0644); err != nil {
		d.Fatalf("error creating test file %s: %v", path, err)
	}
}

type Policies struct {
	RolesV1         []rbacv1.Role
	RoleBindingsV1  []rbacv1.RoleBinding
	ResourceQuotaV1 *corev1.ResourceQuota
	Resources       []policyhierarchyv1.GenericResources
}

// createPolicyNode constructs a PolicyNode based on a Policies struct.
func createPolicyNode(
	name string,
	parent string,
	nodeType policyhierarchyv1.PolicyNodeType,
	policies *Policies) policyhierarchyv1.PolicyNode {
	pn := policynode.NewPolicyNode(name,
		&policyhierarchyv1.PolicyNodeSpec{
			Type:   nodeType,
			Parent: parent,
		})
	if policies == nil {
		return *pn
	}

	pn.Spec.RolesV1 = policies.RolesV1
	pn.Spec.RoleBindingsV1 = policies.RoleBindingsV1
	pn.Spec.ResourceQuotaV1 = policies.ResourceQuotaV1

	if len(pn.Spec.RolesV1) > 0 {
		var roleObjects []runtime.Object
		for _, role := range policies.RolesV1 {
			roleObjects = append(roleObjects, runtime.Object(&role))
		}
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects(roleObjects, rbacv1.SchemeGroupVersion, "Role")...)
	}
	if len(pn.Spec.RoleBindingsV1) > 0 {
		var rbObjects []runtime.Object
		for _, rb := range policies.RoleBindingsV1 {
			rbObjects = append(rbObjects, runtime.Object(&rb))
		}
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects(rbObjects, rbacv1.SchemeGroupVersion, "RoleBinding")...)
	}
	if policies.ResourceQuotaV1 != nil {
		o := runtime.Object(policies.ResourceQuotaV1)
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects([]runtime.Object{o}, corev1.SchemeGroupVersion, "ResourceQuota")...)
	}
	if policies.Resources != nil {
		pn.Spec.Resources = append(pn.Spec.Resources, policies.Resources...)
	}
	return *pn
}

func resourcesFromObjects(objects []runtime.Object, gv schema.GroupVersion, kind string) []policyhierarchyv1.GenericResources {
	raws := []runtime.RawExtension{}
	for _, o := range objects {
		raws = append(raws, runtime.RawExtension{Object: o})
	}
	if len(raws) > 0 {
		res := policyhierarchyv1.GenericResources{
			Group: gv.Group,
			Kind:  kind,
			Versions: []policyhierarchyv1.GenericVersionResources{
				{
					Version: gv.Version,
					Objects: raws,
				},
			},
		}
		return []policyhierarchyv1.GenericResources{res}
	}
	return []policyhierarchyv1.GenericResources{}
}

func createNamespacePN(
	path string,
	parent string,
	policies *Policies) policyhierarchyv1.PolicyNode {
	return createNamespacePNWithLabelsAndAnnotations(path, parent, policies, nil, nil)
}

func createNamespacePNWithLabelsAndAnnotations(
	path string,
	parent string,
	policies *Policies,
	labels, annotations map[string]string,
) policyhierarchyv1.PolicyNode {
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["nomos.dev/declaration-path"] = path
	pn := createPolicyNode(filepath.Base(filepath.Dir(path)), parent, policyhierarchyv1.Namespace, policies)
	pn.Labels = labels
	pn.Annotations = annotations
	return pn
}

func createPolicyspacePN(
	name string,
	parent string,
	policies *Policies) policyhierarchyv1.PolicyNode {
	return createPolicyNode(name, parent, policyhierarchyv1.Policyspace, policies)
}

func createReservedPN(
	name string,
	parent string,
	policies *Policies) policyhierarchyv1.PolicyNode {
	return createPolicyNode(name, parent, policyhierarchyv1.ReservedNamespace, policies)
}

func createClusterPolicy() *policyhierarchyv1.ClusterPolicy {
	return policynode.NewClusterPolicy(policyhierarchyv1.ClusterPolicyName,
		&policyhierarchyv1.ClusterPolicySpec{})
}

func createResourceQuota(path, name, namespace string, labels map[string]string) *corev1.ResourceQuota {
	rq := &corev1.ResourceQuota{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{"pods": resource.MustParse("10")},
		},
	}
	if path != "" {
		rq.ObjectMeta.Annotations = map[string]string{"nomos.dev/declaration-path": path}
	}
	return rq
}

func toIntPointer(i int) *int {
	return &i
}

func toInt32Pointer(i int32) *int32 {
	return &i
}

type fileContentMap map[string]string

type parserTestCase struct {
	testName                   string
	root                       string
	testFiles                  fileContentMap
	expectedPolicyNodes        map[string]policyhierarchyv1.PolicyNode
	expectedNumPolicies        map[string]int
	expectedClusterPolicy      *policyhierarchyv1.ClusterPolicy
	expectedNumClusterPolicies *int
	expectedError              bool
}

var parserTestCases = []parserTestCase{
	{
		testName: "Namespace dir with YAML Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar":  createNamespacePN("tree/bar/ns.yaml", "tree", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with JSON Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.json": templateData{Name: "bar"}.apply(aNamespaceJSON),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar":  createNamespacePN("tree/bar/ns.json", "tree", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with Namespace with labels/annotations",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml": aNamespaceWithLabelsAndAnnotations,
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar": createNamespacePNWithLabelsAndAnnotations("tree/bar/ns.yaml", "tree", nil,
				map[string]string{"env": "prod"}, map[string]string{"audit": "true"}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with ignored files",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/ignore":  "",
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar":  createNamespacePN("tree/bar/ns.yaml", "tree", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with 2 ignored files",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/ignore":  "",
			"tree/bar/ignore2": "blah blah blah",
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar":  createNamespacePN("tree/bar/ns.yaml", "tree", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with multiple Namespaces",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/ns2.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir without Namespace multiple",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ignore":  "",
			"tree/bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with namespace mismatch",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with invalid name",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/baR/ns.yaml": templateData{Name: "baR"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with single ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/rq.yaml":  templateData{Namespace: "bar"}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar": createNamespacePN("tree/bar/ns.yaml", "tree",
				&Policies{
					ResourceQuotaV1: createResourceQuota(
						"tree/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, "bar", resourcequota.NewNomosQuotaLabels()),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "ResourceQuota without declared Sync",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/rq.yaml":  templateData{Namespace: "bar"}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar": createNamespacePN("tree/bar/ns.yaml", "tree",
				&Policies{
					ResourceQuotaV1: createResourceQuota(
						"tree/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, "bar", resourcequota.NewNomosQuotaLabels()),
				},
			),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with single ResourceQuota single file",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":   aNomosConfig,
			"system/rq.yaml":      templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/combo.yaml": templateData{Name: "bar"}.apply(aNamespace) + "\n---\n" + templateData{Namespace: "bar"}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar": createNamespacePN("tree/bar/combo.yaml", "tree",
				&Policies{ResourceQuotaV1: createResourceQuota(
					"tree/bar/combo.yaml", resourcequota.ResourceQuotaObjectName, "bar", resourcequota.NewNomosQuotaLabels()),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with multiple ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/rq.yaml":  templateData{ID: "1", Namespace: "bar"}.apply(aQuota),
			"tree/bar/rq2.yaml": templateData{ID: "2", Namespace: "bar"}.apply(aQuota),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with multiple ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/rq.yaml":       templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/rq.yaml":     templateData{ID: "1"}.apply(aQuota),
			"tree/bar/rq2.yaml":    templateData{ID: "2"}.apply(aQuota),
			"tree/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ResourceQuota namespace mismatch",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/rq.yaml":  templateData{Namespace: "baz"}.apply(aQuota),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with multiple Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":   aNomosConfig,
			"system/role.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"tree/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/role1.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aRole),
			"tree/bar/role2.yaml": templateData{ID: "2", Namespace: "bar"}.apply(aRole),
		},
		expectedNumPolicies: map[string]int{"tree": 0, "bar": 2},
	},
	{
		testName: "Namespace dir with deployment",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":        aNomosConfig,
			"system/depl.yaml":         templateData{Group: "apps", Version: "v1", Kind: "Deployment"}.apply(aSync),
			"tree/bar/ns.yaml":         templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/deployment.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aDeployment),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar": createNamespacePN("tree/bar/ns.yaml", "tree",
				&Policies{Resources: []policyhierarchyv1.GenericResources{
					{
						Group: "apps",
						Kind:  "Deployment",
						Versions: []policyhierarchyv1.GenericVersionResources{
							{
								Version: "v1",
								Objects: []runtime.RawExtension{
									{
										Object: runtime.Object(&appsv1.Deployment{
											ObjectMeta: v1.ObjectMeta{
												Name:      "nginx-deployment",
												Namespace: "bar",
											},
											Spec: appsv1.DeploymentSpec{
												Replicas: toInt32Pointer(3),
											},
										}),
									},
								},
							},
						},
					},
				},
				}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with CRD",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":   aNomosConfig,
			"system/eng.yaml":     templateData{Group: "employees", Version: "v1alpha1", Kind: "Engineer"}.apply(aSync),
			"tree/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/philo.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aPhilo),
		},
		expectedNumPolicies: map[string]int{"tree": 0, "bar": 1},
	},
	{
		testName: "Namespace dir with duplicate Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":   aNomosConfig,
			"system/role.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"tree/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/role1.yaml": templateData{Namespace: "bar"}.apply(aRole),
			"tree/bar/role2.yaml": templateData{Namespace: "bar"}.apply(aRole),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rb.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/r1.yaml":  templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"tree/bar/r2.yaml":  templateData{ID: "2", Namespace: "bar"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"tree": 0, "bar": 2},
	},
	{
		testName: "Namespace dir with duplicate Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rb.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/r1.yaml":  templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"tree/bar/r2.yaml":  templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with duplicate Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/rb.yaml":       templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/bar/ns.yaml":     templateData{Name: "bar"}.apply(aNamespace),
			"tree/bar/r1.yaml":     templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"tree/bar/r2.yaml":     templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"tree/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with non-conflicting reserved Namespace specified",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/reserved.yaml": templateData{Namespace: "baz", Attribute: string(policyhierarchyv1alpha1.ReservedAttribute), Name: policyhierarchyv1alpha1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
			"tree/bar/ns.yaml":     templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"baz":  createReservedPN("baz", "", nil),
			"bar":  createNamespacePN("tree/bar/ns.yaml", "tree", nil),
		},
	},
	{
		testName: "Namespace dir with non-conflicting reserved Namespace, but invalid attribute specified",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/reserved.yaml": templateData{Namespace: "foo", Attribute: "invalid-attribute", Name: policyhierarchyv1alpha1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
			"tree/bar/ns.yaml":     templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with conflicting reserved Namespace specified",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/reserved.yaml": templateData{Namespace: "foo", Attribute: "reserved", Name: policyhierarchyv1alpha1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
			"tree/foo/ns.yaml":     templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "reserved namespace ConfigMap with invalid name",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/reserved.yaml": templateData{Namespace: "foo", Attribute: "reserved", Name: "random-name"}.apply(aConfigMap),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ClusterRole",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"tree/bar/cr.yaml":  templateData{}.apply(aClusterRole),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ClusterRoleBinding",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/crb.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"tree/bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with PodSecurityPolicy",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/psp.yaml":   templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"tree/bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with policyspace child",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns.yaml":    templateData{Name: "baz"}.apply(aNamespace),
			"tree/bar/baz/ignore": "",
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with ignored file",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ignore": "",
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar":  createPolicyspacePN("bar", "tree", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Policyspace dir with ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/rq.yaml":  templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", nil),
			"bar": createPolicyspacePN("bar", "tree",
				&Policies{ResourceQuotaV1: createResourceQuota("tree/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, "", nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Policyspace dir with ResourceQuota namespace set",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/rq.yaml":  templateData{Namespace: "qux"}.apply(aQuota),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":    aNomosConfig,
			"system/rq.yaml":       templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/bar/rq.yaml":     templateData{Namespace: "bar"}.apply(aQuota),
			"tree/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":  aNomosConfig,
			"system/role.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"tree/bar/role.yaml": templateData{}.apply(aRole),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rb.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"tree/bar/rb2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"tree": 0, "bar": 0},
	},
	{
		testName: "Policyspace dir with ClusterRole",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"tree/bar/cr.yaml":  templateData{}.apply(aClusterRole),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with ClusterRoleBinding",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/crb.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"tree/bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with PodSecurityPolicy",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/psp.yaml":   templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"tree/bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with ConfigMap",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/cm.yaml":    templateData{Version: "v1", Kind: "ConfigMap"}.apply(aSync),
			"tree/bar/cm.yaml":  templateData{Namespace: "foo", Attribute: "reserved", Name: policyhierarchyv1alpha1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with NamespaceSelector CRD",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/bar/ns-selector.yaml": aNamespaceSelectorTemplate,
		},
		expectedNumPolicies: map[string]int{"tree": 0, "bar": 0},
	},
	{
		testName: "Policyspace and Namespace dir have duplicate rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml":     aNomosConfig,
			"system/rb.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/bar/rb1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"tree/bar/baz/ns.yaml":  templateData{Name: "baz"}.apply(aNamespace),
			"tree/bar/baz/rb1.yaml": templateData{ID: "1", Namespace: "baz"}.apply(aRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Tree dir empty",
		root:     "foo",
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", &Policies{}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Empty tree and valid Sync",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", &Policies{}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Sync declares multiple versions",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml": `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: ResourceQuota
spec:
  groups:
  - kinds:
    - kind: ResourceQuota
      versions:
      - version: v1
      - version: v2
`,
		},
		expectedError: true,
	},
	{
		testName: "Tree dir with ignore file",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/ignore": "",
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", &Policies{}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Tree dir with Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/ns.yaml": templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Tree dir with ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/rq.yaml":      templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyspacePN("tree", "", &Policies{
				ResourceQuotaV1: createResourceQuota("tree/rq.yaml", resourcequota.ResourceQuotaObjectName, "", nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Tree dir with ResourceQuota and namespace dir",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/role.yaml":  templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"tree/rq.yaml":      templateData{}.apply(aQuota),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]policyhierarchyv1.PolicyNode{
			"tree": createPolicyNode("tree", "", policyhierarchyv1.Policyspace,
				&Policies{ResourceQuotaV1: createResourceQuota("tree/rq.yaml", resourcequota.ResourceQuotaObjectName, "", nil)}),
			"bar": createNamespacePN("tree/bar/ns.yaml", "tree",
				&Policies{ResourceQuotaV1: createResourceQuota(
					"tree/rq.yaml", resourcequota.ResourceQuotaObjectName, "", resourcequota.NewNomosQuotaLabels()),
				}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Tree dir with Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/role.yaml":  templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"tree/role.yaml":    templateData{}.apply(aRole),
		},
		expectedError: true,
	},
	{
		testName: "Tree dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rb.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/r1.yaml":      templateData{ID: "1"}.apply(aRoleBinding),
			"tree/r2.yaml":      templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"tree": 0},
	},
	{
		testName: "Tree dir with multiple inherited Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/rb.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"tree/r1.yaml":      templateData{ID: "1"}.apply(aRoleBinding),
			"tree/r2.yaml":      templateData{ID: "2"}.apply(aRoleBinding),
			"tree/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedNumPolicies: map[string]int{"tree": 0, "bar": 2},
	},
	{
		testName: "Cluster dir with multiple ClusterRoles",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"cluster/cr1.yaml":  templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml":  templateData{ID: "2"}.apply(aClusterRole),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple ClusterRoleBindings",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/crb.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "2"}.apply(aClusterRoleBinding),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple PodSecurityPolicies",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/psp.yaml":   templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "2"}.apply(aPodSecurityPolicy),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with deployment",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/node.yaml":  templateData{Version: "v1", Kind: "Node"}.apply(aSync),
			"cluster/node.yaml": templateData{}.apply(aNode),
		},
		expectedNumClusterPolicies: toIntPointer(1),
	},
	{
		testName: "Cluster dir with duplicate ClusterRole names",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"cluster/cr1.yaml":  templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml":  templateData{ID: "1"}.apply(aClusterRole),
		},
		expectedError: true,
	},
	{
		testName: "Cluster dir with duplicate ClusterRoleBinding names",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/crb.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Cluster dir with duplicate PodSecurityPolicy names",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
			"system/psp.yaml":   templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
		},
		expectedError: true,
	},
	{
		testName: "Dir name not unique 1",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/baz/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"tree/qux/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name not unique 2",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/ns.yaml": templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name not unique 3",
		root:     "foo",
		testFiles: fileContentMap{
			// Two policyspace dirs with same name.
			"tree/bar/baz/corge/ns.yaml": templateData{Name: "corge"}.apply(aNamespace),
			"tree/qux/baz/waldo/ns.yaml": templateData{Name: "waldo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name reserved",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/kube-system/ns.yaml": templateData{Name: "kube-system"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name invalid",
		root:     "foo",
		testFiles: fileContentMap{
			"tree/foo bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name invalid",
		root:     "foo",
		testFiles: fileContentMap{
			"system/nomos.yaml": aNomosConfig,
		},
		expectedNumPolicies: map[string]int{"tree": 0},
	},
}

func TestParser(t *testing.T) {
	for _, tc := range parserTestCases {
		t.Run(tc.testName, func(t *testing.T) {
			d := newTestDir(t, tc.root)
			defer d.remove()

			for k, v := range tc.testFiles {
				d.createTestFile(k, v)
			}

			f := fstesting.NewTestFactory()
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			p := Parser{f, fstesting.TestAPIResourceList(), true}

			actualPolicies, err := p.Parse(d.rootDir)
			if tc.expectedError {
				if err != nil {
					return
				}
				t.Fatalf("Expected error but got none")
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if actualPolicies == nil {
				t.Fatalf("actualPolicies is nil")
			}

			if len(tc.expectedNumPolicies) > 0 {
				n := make(map[string]int)
				for k, v := range actualPolicies.PolicyNodes {
					n[k] = 0
					for _, res := range v.Spec.Resources {
						for _, version := range res.Versions {
							n[k] += len(version.Objects)
						}
					}
				}
				if diff := deep.Equal(n, tc.expectedNumPolicies); diff != nil {
					t.Errorf("Actual and expected number of policy nodes didn't match: %v", diff)
				}
			}

			if tc.expectedNumClusterPolicies != nil {
				p := actualPolicies.ClusterPolicy.Spec
				n := 0
				for _, res := range p.Resources {
					for _, version := range res.Versions {
						n += len(version.Objects)
					}
				}
				if diff := deep.Equal(n, *tc.expectedNumClusterPolicies); diff != nil {
					t.Errorf("Actual and expected number of cluster policies didn't match: %v", diff)
				}
			}

			if tc.expectedPolicyNodes != nil || tc.expectedClusterPolicy != nil {
				expectedPolicies := &policyhierarchyv1.AllPolicies{
					PolicyNodes:   tc.expectedPolicyNodes,
					ClusterPolicy: tc.expectedClusterPolicy,
				}

				if diff := deep.Equal(actualPolicies, expectedPolicies); diff != nil {
					t.Errorf("Actual and expected policies didn't match: %v", diff)
				}
			}
		})
	}
}
