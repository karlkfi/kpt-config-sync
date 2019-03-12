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
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyhierarchy"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	fstesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const (
	objectMetaTemplate = `
{{- if .Annotations}}
  annotations:
  {{range $k, $v := .Annotations}}
    {{$k}}: '{{$v}}'
  {{- end}}
{{- end}}
{{- if .Labels}}
  labels:
  {{- range $k, $v := .Labels}}
    {{$k}}: '{{$v}}'
  {{- end}}
{{- end}}`

	aNamespaceTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Name}}
{{template "objectmetatemplate" .}}
`

	aNamespaceWithLabelsAndAnnotationsTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Name}}
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
{{template "objectmetatemplate" .}}
spec:
  hard:
    pods: "10"
  {{- if .Scope}}
  scopes: ["Terminating"]
  {{- end}}
  {{- if .ScopeSelector}}
  scopeSelector:
    matchExpressions:
      - operator : In
        scopeName: PriorityClass
  {{- end}}
`
	aRoleTemplate = `
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creator{{.ID}}
rules:
- apiGroups: ["batch/v1"]
  resources: ["jobs"]
  verbs:
   - "*"
{{template "objectmetatemplate" .}}
`

	aNamedRoleTemplate = `
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: {{.Name}}
`

	aRoleBindingTemplate = `
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creators{{.ID}}
{{template "objectmetatemplate" .}}
subjects:
- kind: Group
  name: bob@acme.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: job-creator
  apiGroup: rbac.authorization.k8s.io
`

	aLBPRoleBindingTemplate = `
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creators{{.ID}}
  annotations:
    configmanagement.gke.io/namespace-selector: {{.LBPName}}
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
{{template "objectmetatemplate" .}}
rules:
- apiGroups: ["batch/v1"]
  resources: ["jobs"]
  verbs:
   - "*"
`

	// TODO(filmil): factor annotations pipeline out of all objects that use it.
	aClusterRoleBindingTemplate = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: job-creators{{.ID}}
{{template "objectmetatemplate" .}}
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
apiVersion: policy/v1beta1
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
{{template "objectmetatemplate" .}}
`

	aNamespaceSelectorTemplate = `
kind: NamespaceSelector
apiVersion: configmanagement.gke.io/v1
metadata:
  name: sre-supported
{{- if .Namespace}}
{{- end}}
{{template "objectmetatemplate" .}}
spec:
  selector:
    matchLabels:
      environment: prod
`

	aRepo = `
kind: Repo
apiVersion: configmanagement.gke.io/v1
spec:
  version: "0.1.0"
metadata:
  name: repo
`

	aHierarchyConfigTemplate = `
kind: HierarchyConfig
apiVersion: configmanagement.gke.io/v1
metadata:
{{- if .Name}}
  name: {{.Name}}
{{- else}}
  name: {{.KindLower}}
{{- end}}
spec:
  resources:
  - group: {{.Group}}
    kinds: [ {{.Kind}} ]
{{- if .HierarchyMode}}
    hierarchyMode: {{.HierarchyMode}}
{{- end}}
`

	aDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
`

	aPhiloTemplate = `
apiVersion: employees/v1alpha1
kind: Engineer
metadata:
  name: philo
spec:
  cafePreference: 3
`

	aNodeTemplate = `
apiVersion: v1
kind: Node
metadata:
  name: gke-1234
`

	aClusterRegistryClusterTemplate = `
apiVersion: clusterregistry.k8s.io/v1alpha1
kind: Cluster
metadata:
  name: {{.Name}}
{{template "objectmetatemplate" .}}
`

	aClusterSelectorTemplate = `
apiVersion: configmanagement.gke.io/v1
kind: ClusterSelector
metadata:
  name: {{.Name}}
spec:
  selector:
    matchLabels:
      environment: prod
`

	aClusterSelectorWithEnvTemplate = `
apiVersion: configmanagement.gke.io/v1
kind: ClusterSelector
metadata:
  name: {{.Name}}
spec:
  selector:
    matchLabels:
      environment: {{.Environment}}
`
	anUndefinedResourceTemplate = `
apiVersion: does.not.exist/v1
kind: Nonexistent
metadata:
  name: nonexistentname
`
)

func tpl(name, content string) *template.Template {
	// Injects "objectmetatemplate" as a library into the existing template to
	// remove repetition in meta declarations.  There does not seem to be a
	// better way to do this starting from a bunch of strings.
	var b bytes.Buffer
	tpl := template.Must(template.New("lib").Parse(`
{{"{{"}}- define "objectmetatemplate" {{"}}"}}
{{- .ObjectMetaTemplate}}
{{"{{"}}- end{{"}}"}}
{{.Content}}`))
	err := tpl.Execute(&b, struct {
		ObjectMetaTemplate, Content string
	}{
		ObjectMetaTemplate: objectMetaTemplate,
		Content:            content,
	})
	if err != nil {
		panic(err)
	}
	return template.Must(template.New(name).Parse(b.String()))
}

var (
	aNamespace                         = tpl("aNamespace", aNamespaceTemplate)
	aNamespaceWithLabelsAndAnnotations = tpl("aNamespaceWithLabelsAndAnnotations", aNamespaceWithLabelsAndAnnotationsTemplate)
	aNamespaceJSON                     = tpl("aNamespaceJSON", aNamespaceJSONTemplate)
	aQuota                             = tpl("aQuota", aQuotaTemplate)
	aRole                              = tpl("aRole", aRoleTemplate)
	aRoleBinding                       = tpl("aRoleBinding", aRoleBindingTemplate)
	aLBPRoleBinding                    = tpl("aLBPRoleBinding", aLBPRoleBindingTemplate)
	aClusterRole                       = tpl("aClusterRole", aClusterRoleTemplate)
	aClusterRoleBinding                = tpl("aClusterRoleBinding", aClusterRoleBindingTemplate)
	aPodSecurityPolicy                 = tpl("aPodSecurityPolicyTemplate", aPodSecurityPolicyTemplate)
	aConfigMap                         = tpl("aConfigMap", aConfigMapTemplate)
	aHierarchyConfig                   = tpl("aHierarchyConfig", aHierarchyConfigTemplate)
	aPhilo                             = tpl("aPhilo", aPhiloTemplate)
	aNode                              = tpl("aNode", aNodeTemplate)
	aClusterRegistryCluster            = tpl("aClusterRegistryCluster", aClusterRegistryClusterTemplate)
	aClusterSelector                   = tpl("aClusterSelector", aClusterSelectorTemplate)
	aClusterSelectorWithEnv            = tpl("aClusterSelector", aClusterSelectorWithEnvTemplate)
	aNamespaceSelector                 = tpl("aNamespaceSelectorTemplate", aNamespaceSelectorTemplate)
	aNamedRole                         = tpl("aNamedRole", aNamedRoleTemplate)
	anUndefinedResource                = tpl("AnUndefinedResource", anUndefinedResourceTemplate)
)

// templateData can be used to format any of the below values into templates to create
// a repository file set.
type templateData struct {
	ID, Name, Namespace, Attribute string
	Group, Version, Kind           string
	LBPName, HierarchyMode         string
	Labels, Annotations            map[string]string
	// Environment is formatted into selectors that have matchLabels sections.
	Environment          string
	Scope, ScopeSelector bool
}

func (d templateData) apply(t *template.Template) string {
	var b bytes.Buffer
	if err := t.Execute(&b, d); err != nil {
		panic(errors.Wrapf(err, "template data: %#v", d))
	}
	return b.String()
}

func (d templateData) KindLower() string {
	return strings.ToLower(d.Kind)
}

type testDir struct {
	rootDir string
	*testing.T
}

func newTestDir(t *testing.T) *testDir {
	root, err := ioutil.TempDir("", "test_dir")
	if err != nil {
		t.Fatalf("Failed to create test dir %v", err)
	}
	return &testDir{root, t}
}

func (d testDir) remove() {
	os.RemoveAll(d.rootDir)
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

// Functions below produce typed K8S objects based on values in templateData.

func decoder() runtime.Decoder {
	scheme := runtime.NewScheme()
	// Ensure that all API versions we care about are added here.
	corev1.AddToScheme(scheme)
	rbacv1.AddToScheme(scheme)
	v1.AddToScheme(scheme)
	v1.AddToScheme(scheme)

	cf := serializer.NewCodecFactory(scheme)
	return cf.UniversalDeserializer()
}

// mustParse parses a serialized YAML string.
func mustParse(s string, o runtime.Object) {
	if _, _, err := decoder().Decode([]byte(s), nil, o); err != nil {
		panic(errors.Wrapf(err, "while unmarshalling: %q into: %T", s, o))
	}
}

// rb creates a typed role binding from a template.
func rb(d templateData) rbacv1.RoleBinding {
	s := d.apply(aRoleBinding)
	var o rbacv1.RoleBinding
	mustParse(s, &o)
	return o
}

func rbs(ds ...templateData) []rbacv1.RoleBinding {
	var o []rbacv1.RoleBinding
	for _, d := range ds {
		o = append(o, rb(d))
	}
	return o
}

func crbPtr(d templateData) *rbacv1.ClusterRoleBinding {
	s := d.apply(aClusterRoleBinding)
	var o rbacv1.ClusterRoleBinding
	mustParse(s, &o)
	return &o
}

func cfgMapPtr(d templateData) *corev1.ConfigMap {
	s := d.apply(aConfigMap)
	var o corev1.ConfigMap
	mustParse(s, &o)
	return &o
}

type Policies struct {
	RolesV1         []rbacv1.Role
	RoleBindingsV1  []rbacv1.RoleBinding
	ResourceQuotaV1 *corev1.ResourceQuota
	Resources       []v1.GenericResources
}

// createPolicyNode constructs a PolicyNode based on a Policies struct.
func createPolicyNode(name string, policies *Policies) v1.PolicyNode {
	pn := &v1.PolicyNode{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PolicyNode",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PolicyNodeSpec{},
	}
	if policies == nil {
		return *pn
	}

	if len(policies.RolesV1) > 0 {
		var roleObjects []runtime.Object
		for _, role := range policies.RolesV1 {
			roleObjects = append(roleObjects, runtime.Object(&role))
		}
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects(roleObjects, rbacv1.SchemeGroupVersion, "Role")...)
	}
	if len(policies.RoleBindingsV1) > 0 {
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

func resourcesFromObjects(objects []runtime.Object, gv schema.GroupVersion, kind string) []v1.GenericResources {
	var raws []runtime.RawExtension
	for _, o := range objects {
		raws = append(raws, runtime.RawExtension{Object: o})
	}
	if len(raws) > 0 {
		res := v1.GenericResources{
			Group: gv.Group,
			Kind:  kind,
			Versions: []v1.GenericVersionResources{
				{
					Version: gv.Version,
					Objects: raws,
				},
			},
		}
		return []v1.GenericResources{res}
	}
	return []v1.GenericResources{}
}

func createNamespacePN(
	path string,
	policies *Policies) v1.PolicyNode {
	return createPNWithMeta(path, policies, nil, nil)
}

func createPNWithMeta(
	path string,
	policies *Policies,
	labels, annotations map[string]string,
) v1.PolicyNode {
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["configmanagement.gke.io/source-path"] = path
	pn := createPolicyNode(filepath.Base(path), policies)
	pn.Labels = labels
	pn.Annotations = annotations
	return pn
}

// createClusterPolicyWithSpec creates a PolicyNode from the given spec and name.
func createClusterPolicyWithSpec(name string, spec *v1.ClusterPolicySpec) *v1.ClusterPolicy {
	return &v1.ClusterPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterPolicy",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

func createClusterPolicy() *v1.ClusterPolicy {
	return createClusterPolicyWithSpec(v1.ClusterPolicyName, &v1.ClusterPolicySpec{})
}

func createResourceQuota(path, name string, labels map[string]string) *corev1.ResourceQuota {
	rq := &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{"pods": resource.MustParse("10")},
		},
	}
	if path != "" {
		rq.ObjectMeta.Annotations = map[string]string{"configmanagement.gke.io/source-path": path}
	}
	return rq
}

func createDeployment() v1.GenericResources {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-deployment",
			Annotations: map[string]string{
				v1.SourcePathAnnotationKey: "namespaces/bar/deployment.yaml",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: toInt32Pointer(3),
		},
	}
	return v1.GenericResources{
		Group: "apps",
		Kind:  "Deployment",
		Versions: []v1.GenericVersionResources{
			{
				Version: "v1",
				Objects: []runtime.RawExtension{
					{
						Object: runtime.Object(deployment),
					},
				},
			},
		},
	}
}

func toIntPointer(i int) *int {
	return &i
}

func toInt32Pointer(i int32) *int32 {
	return &i
}

func makeSync(group, kind string) v1.Sync {
	s := *v1.NewSync(group, kind)
	s.Finalizers = append(s.Finalizers, v1.SyncFinalizer)
	return s
}

func singleSyncMap(group, kind string) map[string]v1.Sync {
	return syncMap(makeSync(group, kind))
}

func syncMap(syncs ...v1.Sync) map[string]v1.Sync {
	sm := map[string]v1.Sync{}
	for _, sync := range syncs {
		sm[sync.Name] = sync
	}
	return sm
}

type parserTestCase struct {
	testName                   string
	testFiles                  fstesting.FileContentMap
	vet                        bool
	expectedPolicyNodes        map[string]v1.PolicyNode
	expectedNumPolicies        map[string]int
	expectedClusterPolicy      *v1.ClusterPolicy
	expectedNumClusterPolicies *int
	expectedSyncs              map[string]v1.Sync
	expectedErrorCodes         []string
	// Installation side cluster name.
	clusterName string
}

func TestParseRepo(t *testing.T) {
	testCases := []parserTestCase{
		{
			testName:           "missing Repo",
			expectedErrorCodes: []string{vet.MissingRepoErrorCode},
		},
		{
			testName: "Unsupported repo version is an error",
			testFiles: fstesting.FileContentMap{
				"system/repo.yaml": `
kind: Repo
apiVersion: configmanagement.gke.io/v1
spec:
  version: "0.0.0"
metadata:
  name: repo
`,
			},
			expectedErrorCodes: []string{vet.UnsupportedRepoSpecVersionCode},
		},
	}

	for _, tc := range testCases {
		tc.Run(t)
	}
}

var parserTestCases = []parserTestCase{
	{
		testName: "Namespace dir with YAML Namespace",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with JSON Namespace",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.json": templateData{Name: "bar"}.apply(aNamespaceJSON),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with Namespace with labels/annotations",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespaceWithLabelsAndAnnotations),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createPNWithMeta("namespaces/bar", nil,
				map[string]string{"env": "prod"}, map[string]string{"audit": "true"}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with ignored files",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with 2 ignored files",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
			"namespaces/bar/ignore2": "blah blah blah",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Empty namespace dir",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with multiple Namespaces with same name",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ns2.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.MultipleSingletonsErrorCode},
	},
	{
		testName: "Namespace dir with multiple Namespaces with different names",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ns2.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidNamespaceNameErrorCode, vet.MultipleSingletonsErrorCode},
	},
	{
		testName: "Namespace dir with Namespace mismatch and ignored file",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ignore":  "",
			"namespaces/bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidNamespaceNameErrorCode},
	},
	{
		testName: "Namespace dir with namespace mismatch",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidNamespaceNameErrorCode},
	},
	{
		testName: "Namespace dir with invalid name",
		testFiles: fstesting.FileContentMap{
			"namespaces/baR/ns.yaml": templateData{Name: "baR"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidDirectoryNameErrorCode},
	},
	{
		testName: "Namespace dir with single ResourceQuota",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{
					ResourceQuotaV1: createResourceQuota(
						"namespaces/bar/rq.yaml", "pod-quota", nil),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "ResourceQuota without declared HierarchyConfig",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{
					ResourceQuotaV1: createResourceQuota(
						"namespaces/bar/rq.yaml", "pod-quota", nil),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "ResourceQuota with scope and no hierarchical quota",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":         templateData{Kind: "ResourceQuota", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{ID: "1", Scope: true, ScopeSelector: true}.apply(aQuota),
		},
		expectedNumPolicies: map[string]int{
			"bar": 1,
		},
	},
	{
		testName: "ResourceQuota with scope and hierarchical quota",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":         templateData{Kind: "ResourceQuota", HierarchyMode: "hierarchicalQuota"}.apply(aHierarchyConfig),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{ID: "1", Scope: true, ScopeSelector: true}.apply(aQuota),
		},
		expectedErrorCodes: []string{vet.IllegalResourceQuotaFieldErrorCode, vet.IllegalResourceQuotaFieldErrorCode},
	},
	{
		testName: "Namespaces dir with single ResourceQuota single file",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/combo.yaml": templateData{Name: "bar"}.apply(aNamespace) + "\n---\n" + templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{ResourceQuotaV1: createResourceQuota(
					"namespaces/bar/combo.yaml", "pod-quota", nil),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "Namespace dir with multiple Roles",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/role1.yaml": templateData{ID: "1"}.apply(aRole),
			"namespaces/bar/role2.yaml": templateData{ID: "2"}.apply(aRole),
		},
		expectedNumPolicies: map[string]int{"bar": 2},
	},
	{
		testName: "Namespace dir with deployment",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":         templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/deployment.yaml": aDeploymentTemplate,
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{Resources: []v1.GenericResources{
					createDeployment(),
				},
				}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         singleSyncMap("apps", "Deployment"),
	},
	{
		testName: "Namespace dir with CRD",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/philo.yaml": templateData{ID: "1"}.apply(aPhilo),
		},
		expectedNumPolicies: map[string]int{"bar": 1},
	},
	{
		testName: "Namespace dir with duplicate Roles",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/role1.yaml": templateData{}.apply(aRole),
			"namespaces/bar/role2.yaml": templateData{}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Namespace dir with multiple RoleBindings",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"bar": 2},
	},
	{
		testName: "Namespace dir with duplicate RoleBindings",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Abstract Namespace dir with duplicate RoleBindings",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/r1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Abstract Namespace dir with duplicate unmaterialized RoleBindings",
		testFiles: fstesting.FileContentMap{
			"system/repo.yaml":       aRepo,
			"system/rb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding"}.apply(aHierarchyConfig),
			"namespaces/bar/r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
	},
	{
		testName: "Namespace dir with ClusterRole",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/cr.yaml": templateData{}.apply(aClusterRole),
		},
		expectedErrorCodes: []string{vet.IllegalKindInNamespacesErrorCode},
	},
	{
		testName: "Namespace dir with ClusterRoleBinding",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalKindInNamespacesErrorCode},
	},
	{
		testName: "Namespace dir with PodSecurityPolicy",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedErrorCodes: []string{vet.IllegalKindInNamespacesErrorCode},
	},
	{
		testName: "Namespace dir with Abstract Namespace child",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":       templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/baz/role.yaml": templateData{Name: "role"}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.IllegalNamespaceSubdirectoryErrorCode},
	},
	{
		testName: "Abstract Namespace dir with ignored file",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ignore": "",
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Abstract Namespace dir with RoleBinding, default inheritance",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{},
	},
	{
		testName: "Abstract Namespace dir with RoleBinding, hierarchicalQuota mode specified",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":          templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", HierarchyMode: "hierarchicalQuota"}.apply(aHierarchyConfig),
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalHierarchyModeErrorCode},
	},
	{
		testName: "Abstract Namespace dir with RoleBinding, inheritance off",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":          templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Namespaces dir with RoleBinding, inherit specified",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":         templateData{Kind: kinds.RoleBinding().Kind, Group: kinds.RoleBinding().Group, HierarchyMode: "inherit"}.apply(aHierarchyConfig),
			"namespaces/rb.yaml":     templateData{}.apply(aRoleBinding),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{RoleBindingsV1: rbs(templateData{Annotations: map[string]string{
					v1.SourcePathAnnotationKey: "namespaces/rb.yaml",
				}})}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         singleSyncMap(kinds.RoleBinding().Group, kinds.RoleBinding().Kind),
	},
	{
		testName: "Namespaces dir with ResourceQuota, default inheritance",
		testFiles: fstesting.FileContentMap{
			"namespaces/rq.yaml":     templateData{}.apply(aQuota),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{ResourceQuotaV1: createResourceQuota("namespaces/rq.yaml", "pod-quota", nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "Abstract Namespace dir with uninheritable ResourceQuota, inheritance off",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":         templateData{Kind: "ResourceQuota", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Abstract Namespace dir with ResourceQuota, inherit specified",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":         templateData{Kind: "ResourceQuota", HierarchyMode: "inherit"}.apply(aHierarchyConfig),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedPolicyNodes:   map[string]v1.PolicyNode{},
		expectedSyncs:         syncMap(),
	},
	{
		testName: "Abstract Namespace dir with uninheritable Rolebinding",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/rb.yaml": templateData{}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Abstract Namespace dir with NamespaceSelector CRD",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns-selector.yaml": templateData{}.apply(aNamespaceSelector),
		},
		expectedNumPolicies: map[string]int{},
	},
	{
		testName: "Abstract Namespace dir with NamespaceSelector CRD and object",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns-selector.yaml": templateData{}.apply(aNamespaceSelector),
			"namespaces/bar/rb.yaml":          templateData{ID: "1", LBPName: "sre-supported"}.apply(aLBPRoleBinding),
			"namespaces/bar/prod-ns/ns.yaml":  templateData{Name: "prod-ns", Labels: map[string]string{"environment": "prod"}}.apply(aNamespace),
			"namespaces/bar/test-ns/ns.yaml":  templateData{Name: "test-ns"}.apply(aNamespace),
		},
		expectedNumPolicies: map[string]int{"prod-ns": 1, "test-ns": 0},
	},
	{
		testName: "Abstract Namespace and Namespace dir have duplicate RoleBindings",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/rb1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml":  templateData{Name: "baz"}.apply(aNamespace),
			"namespaces/bar/baz/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Abstract Namespace and Namespace dir have duplicate Deployments",
		testFiles: fstesting.FileContentMap{
			"namespaces/depl1.yaml":     aDeploymentTemplate,
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/depl1.yaml": aDeploymentTemplate,
		},
		expectedErrorCodes: []string{
			vet.MetadataNameCollisionErrorCode,
		},
	},
	{
		testName:              "Minimal repo",
		testFiles:             fstesting.FileContentMap{},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Only system dir with valid HierarchyConfig",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml": templateData{Kind: "ResourceQuota"}.apply(aHierarchyConfig),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         syncMap(),
	},
	{
		testName: "Multiple resources with HierarchyConfigs",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":   templateData{Kind: "ResourceQuota"}.apply(aHierarchyConfig),
			"system/role.yaml": templateData{Group: "rbac.authorization.k8s.io", Kind: "Role"}.apply(aHierarchyConfig),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         syncMap(),
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
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         syncMap(),
	},
	{
		testName: "Namespaces dir with ignored file",
		testFiles: fstesting.FileContentMap{
			"namespaces/ignore": "",
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespaces dir with Namespace",
		testFiles: fstesting.FileContentMap{
			"namespaces/ns.yaml": templateData{Name: "namespaces"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalTopLevelNamespaceErrorCode},
	},
	{
		testName: "Namespaces dir with ResourceQuota and hierarchical quota inheritance",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":         templateData{Kind: "ResourceQuota", HierarchyMode: string(v1.HierarchyModeHierarchicalQuota)}.apply(aHierarchyConfig),
			"namespaces/rq.yaml":     templateData{}.apply(aQuota),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			"bar": createNamespacePN("namespaces/bar",
				&Policies{ResourceQuotaV1: createResourceQuota(
					"namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewNomosQuotaLabels()),
				}),
		},
		expectedClusterPolicy: createClusterPolicyWithSpec(
			v1.ClusterPolicyName,
			&v1.ClusterPolicySpec{
				Resources: []v1.GenericResources{
					{
						Group: policyhierarchy.GroupName,
						Kind:  "HierarchicalQuota",
						Versions: []v1.GenericVersionResources{
							{
								Version: "v1",
								Objects: []runtime.RawExtension{
									{
										Object: runtime.Object(
											&v1.HierarchicalQuota{
												TypeMeta: metav1.TypeMeta{
													APIVersion: v1.SchemeGroupVersion.String(),
													Kind:       "HierarchicalQuota",
												},
												ObjectMeta: metav1.ObjectMeta{
													Name: resourcequota.ResourceQuotaHierarchyName,
												},
												Spec: v1.HierarchicalQuotaSpec{
													Hierarchy: v1.HierarchicalQuotaNode{
														Name: "namespaces",
														Type: "abstractNamespace",
														ResourceQuotaV1: createResourceQuota(
															"namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, nil),
														Children: []v1.HierarchicalQuotaNode{
															{
																Name: "bar",
																Type: "namespace",
																ResourceQuotaV1: createResourceQuota(
																	"namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewNomosQuotaLabels()),
															},
														},
													},
												},
											}),
									},
								},
							}}}}}),
		expectedSyncs: syncMap(
			makeSync(policyhierarchy.GroupName, "HierarchicalQuota"),
			makeSync("", "ResourceQuota"),
		),
	},
	{
		testName: "Namespaces dir with multiple inherited Rolebindings",
		testFiles: fstesting.FileContentMap{
			"namespaces/rb1.yaml":    templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/rb2.yaml":    templateData{ID: "2"}.apply(aRoleBinding),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedNumPolicies: map[string]int{"bar": 2},
	},
	{
		testName: "Cluster dir with multiple ClusterRoles",
		testFiles: fstesting.FileContentMap{
			"cluster/cr1.yaml": templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml": templateData{ID: "2"}.apply(aClusterRole),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple ClusterRoleBindings",
		testFiles: fstesting.FileContentMap{
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "2"}.apply(aClusterRoleBinding),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple PodSecurityPolicies",
		testFiles: fstesting.FileContentMap{
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "2"}.apply(aPodSecurityPolicy),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with deployment",
		testFiles: fstesting.FileContentMap{
			"cluster/node.yaml": templateData{}.apply(aNode),
		},
		expectedNumClusterPolicies: toIntPointer(1),
	},
	{
		testName: "Cluster dir with duplicate ClusterRole names",
		testFiles: fstesting.FileContentMap{
			"cluster/cr1.yaml": templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml": templateData{ID: "1"}.apply(aClusterRole),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Cluster dir with duplicate ClusterRoleBinding names",
		testFiles: fstesting.FileContentMap{
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Cluster dir with duplicate PodSecurityPolicy names",
		testFiles: fstesting.FileContentMap{
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Dir name not unique 1",
		testFiles: fstesting.FileContentMap{
			"namespaces/baz/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/qux/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.DuplicateDirectoryNameErrorCode},
	},
	{
		testName: "Dir name not unique 2",
		testFiles: fstesting.FileContentMap{
			// Two Abstract Namespace dirs with same name.

			"namespaces/bar/baz/corge/ns.yaml": templateData{Name: "corge"}.apply(aNamespace),
			"namespaces/qux/baz/waldo/ns.yaml": templateData{Name: "waldo"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.DuplicateDirectoryNameErrorCode},
	},
	{
		testName: "Dir name reserved 1",
		testFiles: fstesting.FileContentMap{
			"namespaces/kube-system/ns.yaml": templateData{Name: "kube-system"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.ReservedDirectoryNameErrorCode},
	},
	{
		testName: "Dir name reserved 2",
		testFiles: fstesting.FileContentMap{
			"namespaces/default/ns.yaml": templateData{Name: "default"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.ReservedDirectoryNameErrorCode},
	},
	{
		testName: "Dir name invalid",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo bar/ns.yaml": templateData{Name: "foo bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidDirectoryNameErrorCode},
	},
	{
		testName: "Namespace with NamespaceSelector label is invalid",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar", Annotations: map[string]string{
				v1.NamespaceSelectorAnnotationKey: "prod"},
			}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalNamespaceAnnotationErrorCode},
	},
	{
		testName: "NamespaceSelector may not have ClusterSelector annotations",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns-selector.yaml": templateData{
				Annotations: map[string]string{
					v1.ClusterSelectorAnnotationKey: "something",
				},
			}.apply(aNamespaceSelector),
		},
		expectedErrorCodes: []string{vet.NamespaceSelectorMayNotHaveAnnotationCode},
	},
	{
		testName: "Namespace-scoped object in cluster/ dir",
		testFiles: fstesting.FileContentMap{
			"cluster/rb.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalKindInClusterErrorCode},
	},
	{
		testName: "Illegal annotation definition is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/rb.yaml": templateData{
				Name: "cluster-1",
				Annotations: map[string]string{
					"configmanagement.gke.io/stuff": "prod",
				},
			}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{
			vet.IllegalAnnotationDefinitionErrorCode,
		},
	},
	{
		testName: "Illegal label definition is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/rb.yaml": templateData{
				Name: "cluster-1",
				Labels: map[string]string{
					"configmanagement.gke.io/stuff": "prod",
				},
			}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{
			vet.IllegalLabelDefinitionErrorCode,
		},
	},
	{
		testName: "Illegal object declaration in system/ is an error",
		testFiles: fstesting.FileContentMap{
			"system/configs.yaml": templateData{Name: "myname"}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.IllegalKindInSystemErrorCode},
	},
	{
		testName: "Duplicate Repo definitions is an error",
		testFiles: fstesting.FileContentMap{
			"system/repo-1.yaml": aRepo,
			"system/repo-2.yaml": aRepo,
		},
		expectedErrorCodes: []string{vet.MultipleSingletonsErrorCode},
	},
	{
		testName: "custom resource w/o a CRD applied",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/undefined.yaml": templateData{}.apply(anUndefinedResource),
			"namespaces/bar/ns.yaml":        templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{status.APIServerErrorCode},
	},
	{
		testName: "Name collision in namespace",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/ns.yaml":   templateData{Name: "foo"}.apply(aNamespace),
			"namespaces/foo/rb-1.yaml": templateData{Name: "alice"}.apply(aRoleBinding),
			"namespaces/foo/rb-2.yaml": templateData{Name: "alice"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "No name collision if types different",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/rb-1.yaml": templateData{Name: "alice"}.apply(aRoleBinding),
			"namespaces/foo/rb-2.yaml": templateData{Name: "alice"}.apply(aQuota),
			"namespaces/foo/ns.yaml":   templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedNumPolicies: map[string]int{
			"foo": 2,
		},
	},
	{
		testName: "Name collision in child node",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/rb-1.yaml":     templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/foo/bar/ns.yaml":   templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/foo/bar/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Name collision in grandchild node",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/rb-1.yaml":         templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/foo/bar/qux/ns.yaml":   templateData{Name: "qux"}.apply(aNamespace),
			"namespaces/foo/bar/qux/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "No name collision in sibling nodes",
		testFiles: fstesting.FileContentMap{
			"namespaces/fox/bar/rb-1.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/fox/qux/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
		},
	},
	{
		testName: "Empty string name is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/bar/role1.yaml": templateData{Name: ""}.apply(aNamedRole),
		},
		expectedErrorCodes: []string{vet.MissingObjectNameErrorCode},
	},
	{
		testName: "No name is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/bar/role1.yaml": templateData{}.apply(aNamedRole),
		},
		expectedErrorCodes: []string{vet.MissingObjectNameErrorCode},
	},
	{
		testName: "Repo outside system/ is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/repo.yaml": aRepo,
		},
		expectedErrorCodes: []string{vet.IllegalSystemResourcePlacementErrorCode},
	},
	{
		testName: "HierarchyConfig outside system/ is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/config.yaml": templateData{Group: "rbac.authorization.k8s.io", Kind: "Role"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.IllegalSystemResourcePlacementErrorCode},
	},
	{
		testName: "HierarchyConfig contains a CRD",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: "extensions", Kind: "CustomResourceDefinition"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "HierarchyConfig contains a Namespace",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Kind: "Namespace"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "HierarchyConfig contains a PolicyNode",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: policyhierarchy.GroupName, Kind: "PolicyNode"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "HierarchyConfig contains a Sync",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: policyhierarchy.GroupName, Kind: "Sync"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "Invalid name for HierarchyConfig",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: "rbac.authorization.k8s.io", Kind: "Role", Name: "RBAC"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.InvalidMetadataNameErrorCode},
	},
	{
		testName: "Illegal Namespace in clusterregistry/",
		testFiles: fstesting.FileContentMap{
			"clusterregistry/namespace.yaml": templateData{Name: "clusterregistry"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalKindInClusterregistryErrorCode},
	},
	{
		testName: "Illegal NamespaceSelector in Namespace directory.",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/namespace.yaml":         templateData{Name: "foo"}.apply(aNamespace),
			"namespaces/foo/namespaceselector.yaml": templateData{}.apply(aNamespaceSelector),
		},
		expectedErrorCodes: []string{vet.IllegalKindInNamespacesErrorCode},
	},
}

func (tc *parserTestCase) Run(t *testing.T) {
	d := newTestDir(t)
	defer d.remove()

	// Used in per-cluster addressing tests.  If undefined should mean
	// the behavior does not change with respect to "regular" state.
	if err := os.Setenv("CLUSTER_NAME", tc.clusterName); err != nil {
		t.Fatal("could not set up CLUSTER_NAME envvar for testing")
	}
	defer os.Unsetenv("CLUSTER_NAME")

	if glog.V(6) {
		glog.Infof("Testcase: %+v", spew.Sdump(tc))
	}

	for k, v := range tc.testFiles {
		// stuff
		d.createTestFile(k, v)
	}

	f := fstesting.NewTestFactory(t)
	defer func() {
		if err := f.Cleanup(); err != nil {
			t.Fatal(errors.Wrap(err, "could not clean up"))
		}
	}()
	p, err := NewParserWithFactory(
		f,
		ParserOpt{
			Vet:       tc.vet,
			Validate:  true,
			Extension: &NomosVisitorProvider{},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	actualPolicies, mErr := p.Parse(d.rootDir, "", time.Time{})

	vettesting.ExpectErrors(tc.expectedErrorCodes, mErr, t)
	if mErr != nil {
		// We expected there to be an error, so no need to do policy validation
		return
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
		if !cmp.Equal(n, tc.expectedNumPolicies) {
			t.Errorf("Actual and expected number of policy nodes didn't match: %v", cmp.Diff(n, tc.expectedNumPolicies))
		}
	}

	if tc.expectedNumClusterPolicies != nil {
		actualNumClusterPolicies := 0
		for _, res := range actualPolicies.ClusterPolicy.Spec.Resources {
			for _, version := range res.Versions {
				actualNumClusterPolicies += len(version.Objects)
			}
		}

		if !cmp.Equal(actualNumClusterPolicies, *tc.expectedNumClusterPolicies) {
			t.Errorf("Actual and expected number of cluster policies didn't match: %v", cmp.Diff(actualNumClusterPolicies,
				tc.expectedNumClusterPolicies))
		}
	}

	if tc.expectedPolicyNodes != nil || tc.expectedClusterPolicy != nil || tc.expectedSyncs != nil {
		if tc.expectedPolicyNodes == nil {
			tc.expectedPolicyNodes = map[string]v1.PolicyNode{}
		}
		if tc.expectedSyncs == nil {
			tc.expectedSyncs = map[string]v1.Sync{}
		}

		expectedPolicies := &policynode.AllPolicies{
			PolicyNodes:   tc.expectedPolicyNodes,
			ClusterPolicy: tc.expectedClusterPolicy,
			Syncs:         tc.expectedSyncs,
		}
		if d := cmp.Diff(expectedPolicies, actualPolicies, resourcequota.ResourceQuantityEqual()); d != "" {
			t.Errorf("Actual and expected policies didn't match: diff\n%v", d)
		}
	}
}

func TestParser(t *testing.T) {
	for _, tc := range parserTestCases {
		tc.testFiles["system/repo.yaml"] = aRepo
		t.Run(tc.testName, tc.Run)
	}
}

// TestParserPerClusterAddressing contains tests cases that use the per-cluster
// addressing feature.  These test cases have been factored out into a separate
// test function since the baseline setup is a bit long, and it gets stenciled
// several times over.
func TestParserPerClusterAddressing(t *testing.T) {
	tests := []parserTestCase{
		{
			// Baseline test case: the selector matches the cluster labels, and
			// all resources are targeted to that selector.  This should yield
			// a set of policy documents that are all present and all fully
			// annotated as appropriate.
			testName:    "Cluster filter, all resources selected",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml": templateData{
					Name: "bar",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{
										{
											Object: runtime.Object(
												&rbacv1.ClusterRoleBinding{
													TypeMeta: metav1.TypeMeta{
														APIVersion: "rbac.authorization.k8s.io/v1",
														Kind:       "ClusterRoleBinding",
													},
													ObjectMeta: metav1.ObjectMeta{
														Name: "job-creators1",
														Annotations: map[string]string{
															v1.ClusterNameAnnotationKey:     "cluster-1",
															v1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
															v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
														},
													},
													Subjects: []rbacv1.Subject{{
														Kind:     "Group",
														APIGroup: "rbac.authorization.k8s.io",
														Name:     "bob@acme.com",
													},
													},
													RoleRef: rbacv1.RoleRef{
														Kind:     "ClusterRole",
														APIGroup: "rbac.authorization.k8s.io",
														Name:     "job-creator",
													},
												},
											),
										},
									},
								},
							},
						},
					},
				}),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				"bar": createPNWithMeta("namespaces/bar",
					&Policies{
						RoleBindingsV1: []rbacv1.RoleBinding{
							{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "rbac.authorization.k8s.io/v1",
									Kind:       "RoleBinding",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "job-creators",
									Annotations: map[string]string{
										v1.ClusterNameAnnotationKey:     "cluster-1",
										v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
										v1.SourcePathAnnotationKey:      "namespaces/bar/rolebinding.yaml",
									},
								},
								Subjects: []rbacv1.Subject{{
									Kind:     "Group",
									APIGroup: "rbac.authorization.k8s.io",
									Name:     "bob@acme.com",
								},
								},
								RoleRef: rbacv1.RoleRef{
									Kind:     "Role",
									APIGroup: "rbac.authorization.k8s.io",
									Name:     "job-creator",
								},
							},
						},
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1.ClusterNameAnnotationKey:     "cluster-1",
						v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
			expectedSyncs: syncMap(
				makeSync(kinds.RoleBinding().Group, kinds.RoleBinding().Kind),
				makeSync(kinds.ClusterRoleBinding().Group, kinds.ClusterRoleBinding().Kind),
			),
		},
		{
			testName:    "Generic resource in Abstract Namespace",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir

				"system/configmap-config.yaml": templateData{
					Kind:          "ConfigMap",
					HierarchyMode: "inherit",
				}.apply(aHierarchyConfig),
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/foo/bar/bar.yaml": templateData{
					Name: "bar",
				}.apply(aNamespace),
				"namespaces/foo/configmap.yaml": templateData{
					Name:      "cfg",
					Namespace: "key",
					Attribute: "value",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aConfigMap),
				"namespaces/foo/configmap2.yaml": templateData{
					Name:      "cfg-excluded",
					Namespace: "key",
					Attribute: "value",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-2",
					},
				}.apply(aConfigMap),
			},
			expectedClusterPolicy: createClusterPolicy(),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				"bar": createPNWithMeta("namespaces/foo/bar",
					&Policies{
						Resources: []v1.GenericResources{
							{
								Group: "",
								Kind:  "ConfigMap",
								Versions: []v1.GenericVersionResources{
									{
										Version: "v1",
										Objects: []runtime.RawExtension{
											{
												Object: runtime.Object(
													cfgMapPtr(templateData{
														Name:      "cfg",
														Namespace: "key",
														Attribute: "value",
														Annotations: map[string]string{
															v1.ClusterNameAnnotationKey:     "cluster-1",
															v1.SourcePathAnnotationKey:      "namespaces/foo/configmap.yaml",
															v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
														},
													}),
												),
											},
										},
									},
								},
							},
						},
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1.ClusterNameAnnotationKey: "cluster-1",
					}),
			},
			expectedSyncs: singleSyncMap(corev1.SchemeGroupVersion.Group, "ConfigMap"),
		},
		{
			// When cluster selector doesn't match, nothing (except for top-level dir) is created.
			testName: "Cluster filter, no resources selected",
			// Note that cluster-2 is not part of the selector.
			clusterName: "cluster-2",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml": templateData{
					Name: "bar",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{}),
			expectedPolicyNodes: map[string]v1.PolicyNode{},
			expectedSyncs:       syncMap(),
		},
		{
			// This shows how a namespace scoped resource doesn't get synced if
			// its selector does not match.
			testName:    "Namespace resource selector does not match",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml": templateData{
					Name: "bar",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				// This role binding is targeted to a different selector.
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-2",
					},
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{
										{
											Object: runtime.Object(
												crbPtr(templateData{
													ID: "1",
													Annotations: map[string]string{
														v1.ClusterNameAnnotationKey:     "cluster-1",
														v1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
														v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
													},
												}),
											),
										},
									},
								},
							},
						},
					},
				}),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				"bar": createPNWithMeta("namespaces/bar",
					&Policies{},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1.ClusterNameAnnotationKey:     "cluster-1",
						v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
			expectedSyncs: syncMap(
				makeSync(kinds.ClusterRoleBinding().Group, kinds.ClusterRoleBinding().Kind),
			),
		},
		{
			testName:    "If namespace is not selected, its resources are not selected either.",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				// Note the whole namespace won't match selector "sel-2".
				"namespaces/bar/bar.yaml": templateData{
					Name: "bar",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-2",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{
										{
											Object: runtime.Object(
												crbPtr(templateData{
													ID: "1",
													Annotations: map[string]string{
														v1.ClusterNameAnnotationKey:     "cluster-1",
														v1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
														v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
													},
												}),
											),
										},
									},
								},
							},
						},
					},
				}),
			expectedPolicyNodes: map[string]v1.PolicyNode{},
			expectedSyncs: syncMap(
				makeSync(kinds.ClusterRoleBinding().Group, kinds.ClusterRoleBinding().Kind),
			),
		},
		{
			testName:    "Cluster resources not matching selector",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml": templateData{
					Name: "bar",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"configmanagement.gke.io/cluster-selector": "sel-2",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				// The cluster-scoped policy with mismatching selector was filtered out.
				&v1.ClusterPolicySpec{}),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				"bar": createPNWithMeta("namespaces/bar",
					&Policies{
						RoleBindingsV1: rbs(
							templateData{
								Name: "job-creators",
								Annotations: map[string]string{
									v1.ClusterNameAnnotationKey:     "cluster-1",
									v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
									v1.SourcePathAnnotationKey:      "namespaces/bar/rolebinding.yaml",
								},
							}),
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1.ClusterNameAnnotationKey:     "cluster-1",
						v1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"configmanagement.gke.io/v1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
			expectedSyncs: syncMap(
				makeSync(kinds.RoleBinding().Group, kinds.RoleBinding().Kind),
			),
		},
		{
			testName:    "Resources without cluster selectors are never filtered out",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml":         templateData{Name: "bar"}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{Name: "role"}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{
										{
											Object: runtime.Object(
												crbPtr(templateData{
													ID: "1",
													Annotations: map[string]string{
														v1.ClusterNameAnnotationKey: "cluster-1",
														v1.SourcePathAnnotationKey:  "cluster/crb1.yaml",
													},
												}),
											),
										},
									},
								},
							},
						},
					},
				}),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				"bar": createPNWithMeta("namespaces/bar",
					&Policies{
						RoleBindingsV1: rbs(
							templateData{Name: "job-creators",
								Annotations: map[string]string{
									v1.ClusterNameAnnotationKey: "cluster-1",
									v1.SourcePathAnnotationKey:  "namespaces/bar/rolebinding.yaml",
								},
							}),
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1.ClusterNameAnnotationKey: "cluster-1",
					}),
			},
			expectedSyncs: syncMap(
				makeSync(kinds.RoleBinding().Group, kinds.RoleBinding().Kind),
				makeSync(kinds.ClusterRoleBinding().Group, kinds.ClusterRoleBinding().Kind),
			),
		},
		{
			// Look at Tree dir below for the meat of the test.
			testName: "Quotas targeted to different clusters may coexist in a namespace",
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/cluster-2.yaml": templateData{
					Name: "cluster-2",
					Labels: map[string]string{
						"environment": "test",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name:        "sel-1",
					Environment: "prod",
				}.apply(aClusterSelectorWithEnv),
				"clusterregistry/sel-2.yaml": templateData{
					Name:        "sel-2",
					Environment: "test",
				}.apply(aClusterSelectorWithEnv),
				// Tree dir  The quota resources below are in the same directory,
				// but targeted to a different cluster.
				"namespaces/bar/quota-1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						v1.ClusterSelectorAnnotationKey: "sel-1",
					},
				}.apply(aQuota),
				"namespaces/bar/quota-2.yaml": templateData{
					ID: "2",
					Annotations: map[string]string{
						v1.ClusterSelectorAnnotationKey: "sel-2",
					},
				}.apply(aQuota),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: createClusterPolicyWithSpec(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					Resources: []v1.GenericResources{
						{
							Group: "rbac.authorization.k8s.io",
							Kind:  "ClusterRoleBinding",
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1",
									Objects: []runtime.RawExtension{
										{
											Object: runtime.Object(
												crbPtr(templateData{
													ID: "1",
													Annotations: map[string]string{
														v1.SourcePathAnnotationKey: "cluster/crb1.yaml",
													},
												}),
											),
										},
									},
								},
							},
						},
					},
				}),
			expectedPolicyNodes: map[string]v1.PolicyNode{},
			expectedSyncs: syncMap(
				makeSync(kinds.ClusterRoleBinding().Group, kinds.ClusterRoleBinding().Kind),
			),
		},
	}
	for _, test := range tests {
		test.testFiles["system/repo.yaml"] = aRepo
		t.Run(test.testName, test.Run)
	}
}

// TestParserPerClusterAddressingVet tests nomos vet validation errors.
func TestParserPerClusterAddressingVet(t *testing.T) {
	tests := []parserTestCase{
		{
			testName:    "An object that has a cluster selector annotation for nonexistent cluster is an error",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml": templateData{Name: "bar"}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						v1.ClusterSelectorAnnotationKey: "unknown-selector",
					},
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			},
			expectedErrorCodes: []string{vet.ObjectHasUnknownClusterSelectorCode},
		},
		{
			testName:    "A cluster object that has a cluster selector annotation for nonexistent cluster is an error",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				// Cluster registry dir
				"clusterregistry/cluster-1.yaml": templateData{
					Name: "cluster-1",
					Labels: map[string]string{
						"environment": "prod",
					},
				}.apply(aClusterRegistryCluster),
				"clusterregistry/sel-1.yaml": templateData{
					Name: "sel-1",
				}.apply(aClusterSelector),
				// Tree dir
				"namespaces/bar/bar.yaml": templateData{Name: "bar"}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
				}.apply(aRoleBinding),
				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{ID: "1",
					Annotations: map[string]string{
						v1.ClusterSelectorAnnotationKey: "unknown-selector",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedErrorCodes: []string{vet.ObjectHasUnknownClusterSelectorCode},
		},
		{
			testName:    "Defining invalid yaml is an error.",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				"namespaces/invalid.yaml": "This is not valid yaml.",
			},
			expectedErrorCodes: []string{status.APIServerErrorCode},
		},
		{
			testName:    "A subdir of system is an error",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				"system/sub/rb.yaml": templateData{
					Kind:          "ConfigMap",
					HierarchyMode: "inherit",
				}.apply(aHierarchyConfig),
			},
			expectedErrorCodes: []string{vet.IllegalSubdirectoryErrorCode},
		},
		{
			testName:    "Objects in non-namespaces/ with an invalid label is an error",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				"system/hc.yaml": `
kind: HierarchyConfig
apiVersion: configmanagement.gke.io/v1
metadata:
  name: hc
  labels:
    configmanagement.gke.io/illegal-label: "true"`,
			},
			expectedErrorCodes: []string{vet.IllegalLabelDefinitionErrorCode},
		},
		{
			testName:    "Objects in non-namespaces/ with an invalid annotation is an error",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				"system/hc.yaml": `
kind: HierarchyConfig
apiVersion: configmanagement.gke.io/v1
metadata:
  name: hc
  annotations:
    configmanagement.gke.io/unsupported: "true"`,
			},
			expectedErrorCodes: []string{vet.IllegalAnnotationDefinitionErrorCode},
		},
	}
	for _, test := range tests {
		test.testFiles["system/repo.yaml"] = aRepo
		t.Run(test.testName, test.Run)
	}
}

func TestEmptyDirectories(t *testing.T) {
	d := newTestDir(t)
	defer d.remove()

	// Create required repo definition.
	d.createTestFile(filepath.Join(repo.SystemDir, "repo.yaml"), aRepo)

	for _, path := range []string{
		filepath.Join(d.rootDir, repo.NamespacesDir),
		filepath.Join(d.rootDir, repo.ClusterDir),
	} {
		t.Run(path, func(t *testing.T) {
			if err := os.MkdirAll(path, 0750); err != nil {
				d.Fatalf("error creating test dir %s: %v", path, err)
			}
			f := fstesting.NewTestFactory(t)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			p, err := NewParserWithFactory(
				f,
				ParserOpt{
					Vet:       false,
					Validate:  true,
					Extension: &NomosVisitorProvider{},
				},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			actualPolicies, mErr := p.Parse(d.rootDir, "", time.Time{})
			if mErr != nil {
				t.Fatalf("unexpected error: %v", mErr)
			}
			expectedPolicies := &policynode.AllPolicies{
				PolicyNodes:   map[string]v1.PolicyNode{},
				ClusterPolicy: createClusterPolicy(),
				Syncs:         map[string]v1.Sync{},
			}
			if !cmp.Equal(actualPolicies, expectedPolicies) {
				t.Errorf("actual and expected AllPolicies didn't match: %v", cmp.Diff(actualPolicies, expectedPolicies))
			}
		})
	}
}
