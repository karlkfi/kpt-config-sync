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
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	fstesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/google/nomos/pkg/resourcequota"
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
    nomos.dev/namespace-selector: {{.LBPName}}
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
{{template "objectmetatemplate" .}}
`

	aNamespaceSelectorTemplate = `
kind: NamespaceSelector
apiVersion: nomos.dev/v1alpha1
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
apiVersion: nomos.dev/v1alpha1
spec:
  version: "0.1.0"
metadata:
  name: repo
`

	aNamedSyncTemplate = `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: {{.Name}}
spec:
  groups:
  - group: {{.Group}}
    kinds:
    - kind: {{.Kind}}
      versions:
      - version: {{.Version}}
`

	aNamespaceSync = `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: sync
spec:
  groups:
  - kinds:
    - kind: Namespace
      versions:
      - version: v1
`

	aSyncTemplate = `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: {{.KindLower}}
spec:
  groups:
  - group: {{.Group}}
    kinds:
    - kind: {{.Kind}}
      versions:
      - version: {{.Version}}
`

	aHierarchicalSyncTemplate = `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: {{.KindLower}}
spec:
  groups:
  - group: {{.Group}}
    kinds:
    - kind: {{.Kind}}
      hierarchyMode: {{.HierarchyMode}}
      versions:
      - version: {{.Version}}
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
apiVersion: nomos.dev/v1alpha1
kind: ClusterSelector
metadata:
  name: {{.Name}}
spec:
  selector:
    matchLabels:
      environment: prod
`

	aClusterSelectorWithEnvTemplate = `
apiVersion: nomos.dev/v1alpha1
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
	aSync                              = tpl("aSync", aSyncTemplate)
	aHierarchicalSync                  = tpl("aHierarchicalSync", aHierarchicalSyncTemplate)
	aPhilo                             = tpl("aPhilo", aPhiloTemplate)
	aNode                              = tpl("aNode", aNodeTemplate)
	aClusterRegistryCluster            = tpl("aClusterRegistryCluster", aClusterRegistryClusterTemplate)
	aClusterSelector                   = tpl("aClusterSelector", aClusterSelectorTemplate)
	aClusterSelectorWithEnv            = tpl("aClusterSelector", aClusterSelectorWithEnvTemplate)
	aNamespaceSelector                 = tpl("aNamespaceSelectorTemplate", aNamespaceSelectorTemplate)
	aNamedRole                         = tpl("aNamedRole", aNamedRoleTemplate)
	aNamedSync                         = tpl("aNamedSync", aNamedSyncTemplate)
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

// Functions below produce typed K8S objects based on values in templateData.

func decoder() runtime.Decoder {
	scheme := runtime.NewScheme()
	// Ensure that all API versions we care about are added here.
	corev1.AddToScheme(scheme)
	rbacv1.AddToScheme(scheme)
	v1.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)

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

func crb(d templateData) rbacv1.ClusterRoleBinding {
	cp := crbPtr(d)
	return *cp
}

func crbPtr(d templateData) *rbacv1.ClusterRoleBinding {
	s := d.apply(aClusterRoleBinding)
	var o rbacv1.ClusterRoleBinding
	mustParse(s, &o)
	return &o
}

func crbs(ds ...templateData) []rbacv1.ClusterRoleBinding {
	var o []rbacv1.ClusterRoleBinding
	for _, d := range ds {
		o = append(o, crb(d))
	}
	return o
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
func createPolicyNode(
	name string,
	parent string,
	nodeType v1.PolicyNodeType,
	policies *Policies) v1.PolicyNode {
	pn := policynode.NewPolicyNode(name,
		&v1.PolicyNodeSpec{
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
	parent string,
	policies *Policies) v1.PolicyNode {
	return createPNWithMeta(path, parent, v1.Namespace, policies, nil, nil)
}

func createPolicyspacePN(
	path string,
	parent string,
	policies *Policies) v1.PolicyNode {
	return createPNWithMeta(path, parent, v1.Policyspace, policies, nil, nil)
}

func createPNWithMeta(
	path string,
	parent string,
	t v1.PolicyNodeType,
	policies *Policies,
	labels, annotations map[string]string,
) v1.PolicyNode {
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["nomos.dev/source-path"] = path
	pn := createPolicyNode(filepath.Base(path), parent, t, policies)
	pn.Labels = labels
	pn.Annotations = annotations
	return pn
}

func createRootPN(
	policies *Policies) v1.PolicyNode {
	pn := createPolicyNode(v1.RootPolicyNodeName, v1.NoParentNamespace, v1.Policyspace, policies)
	pn.Annotations = map[string]string{"nomos.dev/source-path": "namespaces"}
	return pn
}

func createAnnotatedRootPN(policies *Policies, annotations map[string]string) v1.PolicyNode {
	pn := createPolicyNode(v1.RootPolicyNodeName, v1.NoParentNamespace, v1.Policyspace, policies)
	pn.Annotations = annotations
	pn.Annotations[v1alpha1.SourcePathAnnotationKey] = "namespaces"
	return pn
}

func createClusterPolicy() *v1.ClusterPolicy {
	return policynode.NewClusterPolicy(v1.ClusterPolicyName,
		&v1.ClusterPolicySpec{})
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
		rq.ObjectMeta.Annotations = map[string]string{"nomos.dev/source-path": path}
	}
	return rq
}

func createDeployment(ns string) v1.GenericResources {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-deployment",
			Annotations: map[string]string{
				v1alpha1.SourcePathAnnotationKey: "namespaces/bar/deployment.yaml",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: toInt32Pointer(3),
		},
	}
	if ns != "" {
		deployment.ObjectMeta.Namespace = ns
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

func makeSync(group, version, kind string) v1alpha1.Sync {
	return v1alpha1.Sync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nomos.dev/v1alpha1",
			Kind:       "Sync",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       kind,
			Finalizers: []string{v1alpha1.SyncFinalizer},
		},
		Spec: v1alpha1.SyncSpec{
			Groups: []v1alpha1.SyncGroup{
				{
					Group: group,
					Kinds: []v1alpha1.SyncKind{
						{
							Kind: kind,
							Versions: []v1alpha1.SyncVersion{
								{
									Version: version,
								},
							},
						},
					},
				},
			},
		},
	}
}

func mapOfSingleSync(name, group, kind string, versions ...string) map[string]v1alpha1.Sync {
	var sv []v1alpha1.SyncVersion
	for _, v := range versions {
		sv = append(sv, v1alpha1.SyncVersion{Version: v})
	}
	return map[string]v1alpha1.Sync{
		name: {
			TypeMeta: metav1.TypeMeta{
				APIVersion: "nomos.dev/v1alpha1",
				Kind:       "Sync",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Finalizers: []string{v1alpha1.SyncFinalizer},
			},
			Spec: v1alpha1.SyncSpec{
				Groups: []v1alpha1.SyncGroup{
					{
						Group: group,
						Kinds: []v1alpha1.SyncKind{
							{
								Kind:     kind,
								Versions: sv,
							},
						},
					},
				},
			},
		},
	}
}

type parserTestCase struct {
	testName                   string
	root                       string
	testFiles                  fstesting.FileContentMap
	vet                        bool
	expectedPolicyNodes        map[string]v1.PolicyNode
	expectedNumPolicies        map[string]int
	expectedClusterPolicy      *v1.ClusterPolicy
	expectedNumClusterPolicies *int
	expectedSyncs              map[string]v1alpha1.Sync
	expectedErrorCodes         []string
	// Installation side cluster name.
	clusterName string
}

var parserTestCases = []parserTestCase{
	{
		testName: "Namespace dir with YAML Namespace",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar":                 createNamespacePN("namespaces/bar", v1.RootPolicyNodeName, nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Namespace dir with JSON Namespace",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.json": templateData{Name: "bar"}.apply(aNamespaceJSON),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar":                 createNamespacePN("namespaces/bar", v1.RootPolicyNodeName, nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Namespace dir with Namespace with labels/annotations",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespaceWithLabelsAndAnnotations),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createPNWithMeta("namespaces/bar", v1.RootPolicyNodeName, v1.Namespace, nil,
				map[string]string{"env": "prod"}, map[string]string{"audit": "true"}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Namespace dir with ignored files",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar":                 createNamespacePN("namespaces/bar", v1.RootPolicyNodeName, nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Namespace dir with 2 ignored files",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
			"namespaces/bar/ignore2": "blah blah blah",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar":                 createNamespacePN("namespaces/bar", v1.RootPolicyNodeName, nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Empty namespace dir",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar":                 createNamespacePN("namespaces/bar", v1.RootPolicyNodeName, nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Namespace dir with multiple Namespaces with same name",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ns2.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.MultipleNamespacesErrorCode, vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Namespace dir with multiple Namespaces with different names",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ns2.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{
			vet.MultipleNamespacesErrorCode,
			vet.InvalidNamespaceNameErrorCode,
		},
	},
	{
		testName: "Namespace dir without Namespace multiple",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ignore":  "",
			"namespaces/bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidNamespaceNameErrorCode},
	},
	{
		testName: "Namespace dir with namespace mismatch",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidNamespaceNameErrorCode},
	},
	{
		testName: "Namespace dir with invalid name",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/baR/ns.yaml": templateData{Name: "baR"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidDirectoryNameErrorCode},
	},
	{
		testName: "Namespace dir with single ResourceQuota",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createNamespacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{
					ResourceQuotaV1: createResourceQuota(
						"namespaces/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewNomosQuotaLabels()),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "ResourceQuota without declared Sync",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createNamespacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{
					ResourceQuotaV1: createResourceQuota(
						"namespaces/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewNomosQuotaLabels()),
				},
			),
		},
		expectedErrorCodes: []string{vet.UnsyncableNamespaceObjectErrorCode},
	},
	{
		testName: "ResourceQuota with scope and no hierarchical quota",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota", HierarchyMode: "none"}.apply(aHierarchicalSync),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{ID: "1", Scope: true, ScopeSelector: true}.apply(aQuota),
		},
		expectedErrorCodes: nil,
	},
	{
		testName: "ResourceQuota with scope and hierarchical quota",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota", HierarchyMode: "hierarchicalQuota"}.apply(aHierarchicalSync),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{ID: "1", Scope: true, ScopeSelector: true}.apply(aQuota),
		},
		expectedErrorCodes: []string{vet.IllegalResourceQuotaFieldErrorCode, vet.IllegalResourceQuotaFieldErrorCode},
	},
	{
		testName: "Namespace dir with single ResourceQuota single file",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"system/rq.yaml":            templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"namespaces/bar/combo.yaml": templateData{Name: "bar"}.apply(aNamespace) + "\n---\n" + templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createNamespacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{ResourceQuotaV1: createResourceQuota(
					"namespaces/bar/combo.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewNomosQuotaLabels()),
				},
			),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Namespace dir with multiple Roles",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"system/role.yaml":          templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/role1.yaml": templateData{ID: "1"}.apply(aRole),
			"namespaces/bar/role2.yaml": templateData{ID: "2"}.apply(aRole),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 2},
	},
	{
		testName: "Namespace dir with deployment",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":              aRepo,
			"system/depl.yaml":               templateData{Group: "apps", Version: "v1", Kind: "Deployment"}.apply(aSync),
			"namespaces/bar/ns.yaml":         templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/deployment.yaml": aDeploymentTemplate,
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createNamespacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{Resources: []v1.GenericResources{
					createDeployment("bar"),
				},
				}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("Deployment", "apps", "Deployment", "v1"),
	},
	{
		testName: "Namespace dir with CRD",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"system/eng.yaml":           templateData{Group: "employees", Version: "v1alpha1", Kind: "Engineer"}.apply(aSync),
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/philo.yaml": templateData{ID: "1"}.apply(aPhilo),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 1},
	},
	{
		testName: "Namespace dir with duplicate Roles",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"system/role.yaml":          templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/role1.yaml": templateData{}.apply(aRole),
			"namespaces/bar/role2.yaml": templateData{}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Namespace dir with multiple RoleBindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 2},
	},
	{
		testName: "Namespace dir with duplicate RoleBindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Policyspace dir with duplicate RoleBindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":          aRepo,
			"system/rb.yaml":             templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/r1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/r2.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Namespace dir with ClusterRole",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/cr.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/cr.yaml": templateData{}.apply(aClusterRole),
		},
		expectedErrorCodes: []string{vet.UndefinedErrorCode},
	},
	{
		testName: "Namespace dir with ClusterRoleBinding",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/crb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedErrorCodes: []string{vet.UndefinedErrorCode},
	},
	{
		testName: "Namespace dir with PodSecurityPolicy",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/psp.yaml":         templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"namespaces/bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedErrorCodes: []string{vet.UndefinedErrorCode},
	},
	{
		testName: "Namespace dir with policyspace child",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/baz/ignore": "",
		},
		expectedErrorCodes: []string{vet.IllegalNamespaceSubdirectoryErrorCode},
	},
	{
		testName: "Policyspace dir with ignored file",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":     aRepo,
			"namespaces/bar/ignore": "",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar":                 createPolicyspacePN("namespaces/bar", v1.RootPolicyNodeName, nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Policyspace dir with RoleBinding, default inheritance",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/rb.yaml":          templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 0},
	},
	{
		testName: "Policyspace dir with RoleBinding, hierarchicalQuota mode specified",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/rb.yaml":          templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding", HierarchyMode: "hierarchicalQuota"}.apply(aHierarchicalSync),
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalHierarchyModeErrorCode},
	},
	{
		testName: "Policyspace dir with RoleBinding, inheritance off",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/rb.yaml":          templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding", HierarchyMode: "none"}.apply(aHierarchicalSync),
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Policyspace dir with RoleBinding, inherit specified",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota", HierarchyMode: "inherit"}.apply(aHierarchicalSync),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createPolicyspacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{ResourceQuotaV1: createResourceQuota("namespaces/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Policyspace dir with ResourceQuota, default inheritance",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createPolicyspacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{ResourceQuotaV1: createResourceQuota("namespaces/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Policyspace dir with ResourceQuota, hierarchicalQuota mode specified",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota", HierarchyMode: "hierarchicalQuota"}.apply(aHierarchicalSync),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createPolicyspacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{ResourceQuotaV1: createResourceQuota("namespaces/bar/rq.yaml", resourcequota.ResourceQuotaObjectName, nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Policyspace dir with ResourceQuota, inheritance off",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota", HierarchyMode: "none"}.apply(aHierarchicalSync),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Policyspace dir with ResourceQuota, inherit specified",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota", HierarchyMode: "inherit"}.apply(aHierarchicalSync),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Policyspace dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/rb.yaml":          templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/rb2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 0},
	},
	{
		testName: "Policyspace dir with deployment",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":              aRepo,
			"system/depl.yaml":               templateData{Group: "apps", Version: "v1", Kind: "Deployment", HierarchyMode: "inherit"}.apply(aHierarchicalSync),
			"namespaces/bar/deployment.yaml": aDeploymentTemplate,
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(nil),
			"bar": createPolicyspacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{Resources: []v1.GenericResources{createDeployment("")}}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Policyspace dir with ClusterRole",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/cr.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"namespaces/bar/cr.yaml": templateData{}.apply(aClusterRole),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Policyspace dir with ClusterRoleBinding",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/crb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"namespaces/bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Policyspace dir with PodSecurityPolicy",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":       aRepo,
			"system/psp.yaml":         templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"namespaces/bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Policyspace dir with NamespaceSelector CRD",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":               aRepo,
			"namespaces/bar/ns-selector.yaml": templateData{}.apply(aNamespaceSelector),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 0},
	},
	{
		testName: "Policyspace dir with NamespaceSelector CRD and object",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":               aRepo,
			"system/crb.yaml":                 templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/ns-selector.yaml": templateData{}.apply(aNamespaceSelector),
			"namespaces/bar/rb.yaml":          templateData{ID: "1", LBPName: "sre-supported"}.apply(aLBPRoleBinding),
			"namespaces/bar/prod-ns/ns.yaml":  templateData{Name: "prod-ns", Labels: map[string]string{"environment": "prod"}}.apply(aNamespace),
			"namespaces/bar/test-ns/ns.yaml":  templateData{Name: "test-ns"}.apply(aNamespace),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 0, "prod-ns": 1, "test-ns": 0},
	},
	{
		testName: "Policyspace and Namespace dir have duplicate RoleBindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":           aRepo,
			"system/rb.yaml":              templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/bar/rb1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml":  templateData{Name: "baz"}.apply(aNamespace),
			"namespaces/bar/baz/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Policyspace and Namespace dir have duplicate Deployments",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"system/depl.yaml":          templateData{Group: "apps", Version: "v1", Kind: "Deployment"}.apply(aHierarchicalSync),
			"namespaces/depl1.yaml":     aDeploymentTemplate,
			"namespaces/bar/ns.yaml":    templateData{Name: "baz"}.apply(aNamespace),
			"namespaces/bar/depl1.yaml": aDeploymentTemplate,
		},
		expectedErrorCodes: []string{
			vet.MetadataNameCollisionErrorCode,
			vet.IllegalAbstractNamespaceObjectKindErrorCode,
			vet.InvalidNamespaceNameErrorCode,
		},
	},
	{
		testName: "Minimal repo",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
		},
		expectedPolicyNodes:   map[string]v1.PolicyNode{},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Only system dir with valid sync",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
		},
		expectedPolicyNodes:   map[string]v1.PolicyNode{},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Multiple Syncs",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/rq.yaml":    templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"system/role.yaml":  templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
		},
		expectedPolicyNodes:   map[string]v1.PolicyNode{},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs: map[string]v1alpha1.Sync{
			"ResourceQuota": makeSync("", "v1", "ResourceQuota"),
			"Role":          makeSync("rbac.authorization.k8s.io", "v1", "Role"),
		},
	},
	{
		testName: "Sync declares multiple versions",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/rq.yaml": `
kind: Sync
apiVersion: nomos.dev/v1alpha1
metadata:
  name: resource-quota
spec:
  groups:
  - kinds:
    - kind: ResourceQuota
      versions:
      - version: v1
      - version: v2
`,
		},
		expectedErrorCodes: []string{
			vet.DuplicateSyncGroupKindErrorCode,
			vet.UnknownResourceVersionInSyncErrorCode,
		},
	},
	{
		testName: "Namespaces dir with ignore file",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"namespaces/ignore": "",
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(&Policies{}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         map[string]v1alpha1.Sync{},
	},
	{
		testName: "Namespaces dir with Namespace",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":  aRepo,
			"namespaces/ns.yaml": templateData{Name: "namespaces"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalTopLevelNamespaceErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Namespaces dir with ResourceQuota",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":  aRepo,
			"system/rq.yaml":     templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"namespaces/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(&Policies{
				ResourceQuotaV1: createResourceQuota("namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, nil)}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Namespaces dir with ResourceQuota and namespace dir",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rq.yaml":         templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"namespaces/rq.yaml":     templateData{}.apply(aQuota),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]v1.PolicyNode{
			v1.RootPolicyNodeName: createRootPN(
				&Policies{ResourceQuotaV1: createResourceQuota("namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, nil)}),
			"bar": createNamespacePN("namespaces/bar", v1.RootPolicyNodeName,
				&Policies{ResourceQuotaV1: createResourceQuota(
					"namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewNomosQuotaLabels()),
				}),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedSyncs:         mapOfSingleSync("ResourceQuota", "", "ResourceQuota", "v1"),
	},
	{
		testName: "Namespaces dir with Roles",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":    aRepo,
			"system/role.yaml":     templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"namespaces/role.yaml": templateData{}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Namespaces dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":  aRepo,
			"system/rb.yaml":     templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/r2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0},
	},
	{
		testName: "Namespaces dir with multiple inherited Rolebindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":      aRepo,
			"system/rb.yaml":         templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/r1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/r2.yaml":     templateData{ID: "2"}.apply(aRoleBinding),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedNumPolicies: map[string]int{v1.RootPolicyNodeName: 0, "bar": 2},
	},
	{
		testName: "Cluster dir with multiple ClusterRoles",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"cluster/cr1.yaml":  templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml":  templateData{ID: "2"}.apply(aClusterRole),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple ClusterRoleBindings",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/crb.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "2"}.apply(aClusterRoleBinding),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple PodSecurityPolicies",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/psp.yaml":   templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "2"}.apply(aPodSecurityPolicy),
		},
		expectedNumClusterPolicies: toIntPointer(2),
	},
	{
		testName: "Cluster dir with deployment",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/node.yaml":  templateData{Version: "v1", Kind: "Node"}.apply(aSync),
			"cluster/node.yaml": templateData{}.apply(aNode),
		},
		expectedNumClusterPolicies: toIntPointer(1),
	},
	{
		testName: "Cluster dir with duplicate ClusterRole names",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"cluster/cr1.yaml":  templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml":  templateData{ID: "1"}.apply(aClusterRole),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Cluster dir with duplicate ClusterRoleBinding names",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/crb.yaml":   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Clusterregistry dir with duplicate Cluster names",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/cr.yaml":    templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}.apply(aSync),
			"clusterregistry/cluster-1.yaml": templateData{
				Name: "cluster",
				Labels: map[string]string{
					"environment": "prod",
				},
			}.apply(aClusterRegistryCluster),
			"clusterregistry/cluster-2.yaml": templateData{
				Name: "cluster",
				Labels: map[string]string{
					"environment": "prod",
				},
			}.apply(aClusterRegistryCluster),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Cluster dir with duplicate PodSecurityPolicy names",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/psp.yaml":   templateData{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"}.apply(aSync),
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode, vet.UndefinedErrorCode},
	},
	{
		testName: "Dir name not unique 1",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":          aRepo,
			"namespaces/baz/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/qux/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.DuplicateDirectoryNameErrorCode},
	},
	{
		testName: "Dir name not unique 2",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			// Two policyspace dirs with same name.
			"system/nomos.yaml":                aRepo,
			"namespaces/bar/baz/corge/ns.yaml": templateData{Name: "corge"}.apply(aNamespace),
			"namespaces/qux/baz/waldo/ns.yaml": templateData{Name: "waldo"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.DuplicateDirectoryNameErrorCode},
	},
	{
		testName: "Dir name reserved 1",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":              aRepo,
			"namespaces/kube-system/ns.yaml": templateData{Name: "kube-system"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.ReservedDirectoryNameErrorCode, vet.ReservedDirectoryNameErrorCode},
	},
	{
		testName: "Dir name reserved 2",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":              aRepo,
			"namespaces/kube-system/ns.yaml": templateData{Name: "default"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.ReservedDirectoryNameErrorCode, vet.ReservedDirectoryNameErrorCode, vet.InvalidNamespaceNameErrorCode},
	},
	{
		testName: "Dir name reserved 3",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":              aRepo,
			"namespaces/kube-system/ns.yaml": templateData{Name: "nomos-system"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.ReservedDirectoryNameErrorCode, vet.ReservedDirectoryNameErrorCode, vet.InvalidNamespaceNameErrorCode},
	},
	{
		testName: "Dir name invalid",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":          aRepo,
			"namespaces/foo bar/ns.yaml": templateData{Name: "foo bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.InvalidDirectoryNameErrorCode},
	},
	{
		testName: "Namespace with NamespaceSelector label is invalid",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"namespaces/bar/ns.yaml": templateData{Name: "bar", Annotations: map[string]string{
				v1alpha1.NamespaceSelectorAnnotationKey: "prod"},
			}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalNamespaceAnnotationErrorCode},
	},
	{
		testName: "NamespaceSelector may not have ClusterSelector annotations",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"namespaces/bar/ns-selector.yaml": templateData{
				Annotations: map[string]string{
					v1alpha1.ClusterSelectorAnnotationKey: "something",
				},
			}.apply(aNamespaceSelector),
		},
		expectedErrorCodes: []string{vet.NamespaceSelectorMayNotHaveAnnotationCode},
	},
	{
		testName: "Unsyncable cluster object",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"cluster/rb.yaml":   templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.UnsyncableClusterObjectErrorCode},
	},
	{
		testName: "Illegal annotation definition is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"namespaces/rb.yaml": templateData{
				Name: "cluster-1",
				Annotations: map[string]string{
					"nomos.dev/stuff": "prod",
				},
			}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{
			vet.IllegalAnnotationDefinitionErrorCode,
			vet.UnsyncableNamespaceObjectErrorCode,
			vet.IllegalAbstractNamespaceObjectKindErrorCode,
		},
	},
	{
		testName: "Illegal label definition is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"namespaces/rb.yaml": templateData{
				Name: "cluster-1",
				Labels: map[string]string{
					"nomos.dev/stuff": "prod",
				},
			}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{
			vet.IllegalLabelDefinitionErrorCode,
			vet.UnsyncableNamespaceObjectErrorCode,
			vet.IllegalAbstractNamespaceObjectKindErrorCode,
		},
	},
	{
		testName: "Illegal namespace sync declaration is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/syncs.yaml": aNamespaceSync,
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		testName: "Illegal object declaration in system/ is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/syncs.yaml": templateData{Name: "myname"}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.IllegalKindInSystemErrorCode},
	},
	{
		testName: "Duplicate Repo definitions is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos-1.yaml": aRepo,
			"system/nomos-2.yaml": aRepo,
		},
		expectedErrorCodes: []string{vet.MultipleRepoDefinitionsErrorCode, vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Unsupported repo version is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": `
kind: Repo
apiVersion: nomos.dev/v1alpha1
spec:
  version: "0.0.0"
`,
		},
		expectedErrorCodes: []string{vet.UnsupportedRepoSpecVersionCode, vet.MissingObjectNameErrorCode},
	},
	{
		testName: "Sync contains resource w/o a CRD applied",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":             aRepo,
			"system/unknown.yaml":           templateData{Group: "does.not.exist", Version: "v1", Kind: "Nonexistent"}.apply(aSync),
			"namespaces/bar/undefined.yaml": templateData{}.apply(anUndefinedResource),
			"namespaces/bar/ns.yaml":        templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.UnknownResourceInSyncErrorCode},
	},
	{
		testName: "Name collision in node",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":        aRepo,
			"system/rb.yaml":           templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/foo/rb-1.yaml": templateData{Name: "alice"}.apply(aRoleBinding),
			"namespaces/foo/rb-2.yaml": templateData{Name: "alice"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "No name collision if types different",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":        aRepo,
			"system/rb.yaml":           templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"system/rq.yaml":           templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
			"namespaces/foo/rb-1.yaml": templateData{Name: "alice"}.apply(aRoleBinding),
			"namespaces/foo/rb-2.yaml": templateData{Name: "alice"}.apply(aQuota),
		},
	},
	{
		testName: "Name collision in child node",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":            aRepo,
			"system/rb.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/foo/rb-1.yaml":     templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/foo/bar/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Name collision in grandchild node",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":                aRepo,
			"system/rb.yaml":                   templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/foo/rb-1.yaml":         templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/foo/bar/qux/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "No name collision in sibling nodes",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":            aRepo,
			"system/rb.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
			"namespaces/fox/bar/rb-1.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/fox/qux/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
		},
	},
	{
		testName: "Empty string name is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":            aRepo,
			"system/rb.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"namespaces/foo/bar/rb-1.yaml": templateData{Name: ""}.apply(aNamedRole),
		},
		expectedErrorCodes: []string{vet.MissingObjectNameErrorCode, vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "No name is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":            aRepo,
			"system/rb.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
			"namespaces/foo/bar/rb-1.yaml": templateData{}.apply(aNamedRole),
		},
		expectedErrorCodes: []string{vet.MissingObjectNameErrorCode, vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Name collision in system/ is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/rb-1.yaml":  templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding", Name: "sync"}.apply(aNamedSync),
			"system/rb-2.yaml":  templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role", Name: "sync"}.apply(aNamedSync),
		},
		expectedErrorCodes: []string{vet.MetadataNameCollisionErrorCode},
	},
	{
		testName: "Repo outside system/ is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"namespaces/foo/nomos.yaml": aRepo,
		},
		expectedErrorCodes: []string{vet.IllegalSystemResourcePlacementErrorCode, vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Sync outside system/ is an error",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":         aRepo,
			"namespaces/foo/nomos.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
		},
		expectedErrorCodes: []string{vet.IllegalSystemResourcePlacementErrorCode, vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Sync contains a CRD",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/sync.yaml":  templateData{Group: "extensions", Version: "v1beta1", Kind: "CustomResourceDefinition"}.apply(aSync),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		testName: "Sync contains a Namespace",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/sync.yaml":  templateData{Version: "v1", Kind: "Namespace"}.apply(aSync),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		testName: "Sync contains a PolicyNode",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/sync.yaml":  templateData{Group: "nomos.dev", Version: "v1", Kind: "PolicyNode"}.apply(aSync),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInSyncErrorCode, vet.UnknownResourceInSyncErrorCode},
	},
	{
		testName: "Sync contains a Sync",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/sync.yaml":  templateData{Group: "nomos.dev", Version: "v1alpha1", Kind: "Sync"}.apply(aSync),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		testName: "Invalid name for CRD",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml": aRepo,
			"system/sync.yaml":  templateData{Group: "nomos.dev", Version: "v1alpha1", Kind: "Sync", Name: "Sync"}.apply(aNamedSync),
		},
		expectedErrorCodes: []string{vet.InvalidMetadataNameErrorCode, vet.UnsupportedResourceInSyncErrorCode},
	},
	{
		testName: "Illegal Namespace in clusterregistry/",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":              aRepo,
			"clusterregistry/namespace.yaml": templateData{Name: "clusterregistry"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalKindInClusterregistryErrorCode},
	},
	{
		testName: "Illegal NamespaceSelector in namespaces/",
		root:     "foo",
		testFiles: fstesting.FileContentMap{
			"system/nomos.yaml":                     aRepo,
			"namespaces/foo/namespace.yaml":         templateData{Name: "foo"}.apply(aNamespace),
			"namespaces/foo/namespaceselector.yaml": templateData{}.apply(aNamespaceSelector),
		},
		expectedErrorCodes: []string{vet.IllegalKindInNamespacesErrorCode},
	},
}

func (tc *parserTestCase) Run(t *testing.T) {
	d := newTestDir(t, tc.root)
	defer d.remove()

	// Used in per-cluster addressing tests.  If undefined should mean
	// the behavior does not change with respect to "regular" state.
	os.Setenv("CLUSTER_NAME", tc.clusterName)
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
			Extension: ParserConfigFactory(),
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %#v", err)
	}

	actualPolicies, err := p.Parse(d.rootDir, "", time.Time{})

	vettesting.ExpectErrors(tc.expectedErrorCodes, err, t)
	if err != nil {
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
		p := actualPolicies.ClusterPolicy.Spec
		n := 0
		for _, res := range p.Resources {
			for _, version := range res.Versions {
				n += len(version.Objects)
			}
		}
		if !cmp.Equal(n, *tc.expectedNumClusterPolicies) {
			t.Errorf("Actual and expected number of cluster policies didn't match: %v", cmp.Diff(n, *tc.expectedNumClusterPolicies))
		}

		if tc.expectedPolicyNodes != nil || tc.expectedClusterPolicy != nil || tc.expectedSyncs != nil {
			expectedPolicies := &v1.AllPolicies{
				PolicyNodes:   tc.expectedPolicyNodes,
				ClusterPolicy: tc.expectedClusterPolicy,
				Syncs:         tc.expectedSyncs,
			}
			if !cmp.Equal(actualPolicies, expectedPolicies) {
				t.Errorf("Actual and expected policies didn't match: diff\n%v", cmp.Diff(actualPolicies, expectedPolicies))
			}
		}
	}
}

func TestParser(t *testing.T) {
	for _, tc := range parserTestCases {
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
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					ClusterRoleBindingsV1: []rbacv1.ClusterRoleBinding{
						{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "rbac.authorization.k8s.io/v1",
								Kind:       "ClusterRoleBinding",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "job-creators1",
								Annotations: map[string]string{
									v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
									v1alpha1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
									v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
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
					},
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
															v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
															v1alpha1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
															v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
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
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
				"bar": createPNWithMeta("namespaces/bar", v1.RootPolicyNodeName, v1.Namespace,
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
										v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
										v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
										v1alpha1.SourcePathAnnotationKey:      "namespaces/bar/rolebinding.yaml",
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
						v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
						v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
		},
		{
			testName:    "Generic resource in abstract namespace",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml": aRepo,
				"system/configmap-sync.yaml": templateData{
					Group: "", Version: "v1", Kind: "ConfigMap",
					HierarchyMode: "inherit",
				}.apply(aHierarchicalSync),

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
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aConfigMap),
				"namespaces/foo/configmap2.yaml": templateData{
					Name:      "cfg-excluded",
					Namespace: "key",
					Attribute: "value",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-2",
					},
				}.apply(aConfigMap),
			},
			expectedPolicyNodes: map[string]v1.PolicyNode{
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
				"bar": createPNWithMeta("namespaces/foo/bar", v1.RootPolicyNodeName, v1.Namespace,
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
															v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
															v1alpha1.SourcePathAnnotationKey:      "namespaces/foo/configmap.yaml",
															v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
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
						v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
						v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
		},
		{
			// When cluster selector doesn't match, nothing (except for top-level dir) is created.
			testName: "Cluster filter, no resources selected",
			root:     "foo",
			// Note that cluster-2 is not part of the selector.
			clusterName: "cluster-2",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{}),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-2",
					}),
			},
		},
		{
			// This shows how a namespace scoped resource doesn't get synced if
			// its selector does not match.
			testName:    "Namespace resource selector does not match",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				// This role binding is targeted to a different selector.
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-2",
					},
				}.apply(aRoleBinding),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					ClusterRoleBindingsV1: crbs(
						templateData{
							Name: "1",
							Annotations: map[string]string{
								v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
								v1alpha1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
								v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
							},
						},
					),
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
													Name: "1",
													Annotations: map[string]string{
														v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
														v1alpha1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
														v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
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
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
				"bar": createPNWithMeta("namespaces/bar", v1.RootPolicyNodeName, v1.Namespace,
					&Policies{},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
						v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
		},
		{
			testName:    "If namespace is not selected, its resources are not selected either.",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
						"nomos.dev/cluster-selector": "sel-2",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					ClusterRoleBindingsV1: []rbacv1.ClusterRoleBinding{
						{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "rbac.authorization.k8s.io/v1",
								Kind:       "ClusterRoleBinding",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "job-creators1",
								Annotations: map[string]string{
									v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
									v1alpha1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
									v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
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
					},
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
													Name: "1",
													Annotations: map[string]string{
														v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
														v1alpha1.SourcePathAnnotationKey:      "cluster/crb1.yaml",
														v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
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
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
			},
		},
		{
			testName:    "Cluster resources not matching selector",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aNamespace),
				"namespaces/bar/rolebinding.yaml": templateData{
					Name: "role",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-1",
					},
				}.apply(aRoleBinding),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{
					ID: "1",
					Annotations: map[string]string{
						"nomos.dev/cluster-selector": "sel-2",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				// The cluster-scoped policy with mismatching selector was filtered out.
				&v1.ClusterPolicySpec{}),
			expectedPolicyNodes: map[string]v1.PolicyNode{
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
				"bar": createPNWithMeta("namespaces/bar", v1.RootPolicyNodeName, v1.Namespace,
					&Policies{
						RoleBindingsV1: rbs(
							templateData{
								Name: "job-creators",
								Annotations: map[string]string{
									v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
									v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
									v1alpha1.SourcePathAnnotationKey:      "namespaces/bar/rolebinding.yaml",
								},
							}),
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey:     "cluster-1",
						v1alpha1.ClusterSelectorAnnotationKey: `{"kind":"ClusterSelector","apiVersion":"nomos.dev/v1alpha1","metadata":{"name":"sel-1","creationTimestamp":null},"spec":{"selector":{"matchLabels":{"environment":"prod"}}}}`,
					}),
			},
		},
		{
			testName:    "Resources without cluster selectors are never filtered out",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					ClusterRoleBindingsV1: crbs(
						templateData{
							Name: "1",
							Annotations: map[string]string{
								v1alpha1.ClusterNameAnnotationKey: "cluster-1",
								v1alpha1.SourcePathAnnotationKey:  "cluster/crb1.yaml",
							},
						}),
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
													Name: "1",
													Annotations: map[string]string{
														v1alpha1.ClusterNameAnnotationKey: "cluster-1",
														v1alpha1.SourcePathAnnotationKey:  "cluster/crb1.yaml",
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
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
				"bar": createPNWithMeta("namespaces/bar", v1.RootPolicyNodeName, v1.Namespace,
					&Policies{
						RoleBindingsV1: rbs(
							templateData{Name: "job-creators",
								Annotations: map[string]string{
									v1alpha1.ClusterNameAnnotationKey: "cluster-1",
									v1alpha1.SourcePathAnnotationKey:  "namespaces/bar/rolebinding.yaml",
								},
							}),
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
			},
		},
		{
			// Look at Tree dir below for the meat of the test.
			testName: "Quotas targeted to different clusters may coexist in a namespace",
			root:     "foo",
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/rq.yaml":                 templateData{Version: "v1", Kind: "ResourceQuota"}.apply(aSync),
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}.apply(aSync),

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
						v1alpha1.ClusterSelectorAnnotationKey: "sel-1",
					},
				}.apply(aQuota),
				"namespaces/bar/quota-2.yaml": templateData{
					ID: "2",
					Annotations: map[string]string{
						v1alpha1.ClusterSelectorAnnotationKey: "sel-2",
					},
				}.apply(aQuota),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			},
			expectedClusterPolicy: policynode.NewClusterPolicy(
				v1.ClusterPolicyName,
				&v1.ClusterPolicySpec{
					ClusterRoleBindingsV1: crbs(
						templateData{
							Name: "1",
							Annotations: map[string]string{
								v1alpha1.ClusterNameAnnotationKey: "cluster-1",
								v1alpha1.SourcePathAnnotationKey:  "cluster/crb1.yaml",
							},
						}),
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
													Name: "1",
													Annotations: map[string]string{
														v1alpha1.ClusterNameAnnotationKey: "cluster-1",
														v1alpha1.SourcePathAnnotationKey:  "cluster/crb1.yaml",
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
				v1.RootPolicyNodeName: createAnnotatedRootPN(&Policies{},
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
				"bar": createPNWithMeta("namespaces/bar", v1.RootPolicyNodeName, v1.Namespace,
					&Policies{
						RoleBindingsV1: rbs(
							templateData{Name: "job-creators",
								Annotations: map[string]string{
									v1alpha1.ClusterNameAnnotationKey: "cluster-1",
									v1alpha1.SourcePathAnnotationKey:  "namespaces/bar/rolebinding.yaml",
								},
							}),
					},
					/* Labels */
					nil,
					/* Annotations */
					map[string]string{
						v1alpha1.ClusterNameAnnotationKey: "cluster-1",
					}),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, test.Run)
	}
}

// TestParserPerClusterAddressingVet tests nomos vet validation errors.
func TestParserPerClusterAddressingVet(t *testing.T) {
	tests := []parserTestCase{
		{
			testName:    "An object that has a cluster selector annotation for nonexistent cluster is an error",
			root:        "foo",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role", Name: "RoleSync"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding", Name: "RoleBindingSync"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding", Name: "ClusterRoleBindingSync"}.apply(aSync),

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
						v1alpha1.ClusterSelectorAnnotationKey: "unknown-selector",
					},
				}.apply(aRoleBinding),

				// Cluster dir (cluster scoped objects).
				"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			},
			expectedErrorCodes: []string{vet.ObjectHasUnknownClusterSelectorCode},
		},
		{
			testName:    "A cluster object that has a cluster selector annotation for nonexistent cluster is an error",
			root:        "foo",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				// System dir
				"system/nomos.yaml":              aRepo,
				"system/role.yaml":               templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role", Name: "RoleSync"}.apply(aSync),
				"system/rolebinding.yaml":        templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding", Name: "RoleBindingSync"}.apply(aSync),
				"system/clusterrolebinding.yaml": templateData{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding", Name: "ClusterRoleBindingSync"}.apply(aSync),

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
						v1alpha1.ClusterSelectorAnnotationKey: "unknown-selector",
					},
				}.apply(aClusterRoleBinding),
			},
			expectedErrorCodes: []string{vet.ObjectHasUnknownClusterSelectorCode},
		},
		{
			testName:           "A dir with no system directory is an error",
			root:               "foo",
			clusterName:        "cluster-1",
			vet:                true,
			testFiles:          fstesting.FileContentMap{},
			expectedErrorCodes: []string{vet.MissingDirectoryErrorCode, vet.MissingRepoErrorCode},
		},
		{
			testName:    "A system directory defining no Repo object is an error",
			root:        "foo",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				"system/nomos.yaml": "",
			},
			expectedErrorCodes: []string{vet.MissingRepoErrorCode},
		},
		{
			testName:    "Defining invalid yaml is an error.",
			root:        "foo",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				"system/nomos.yaml":       aRepo,
				"namespaces/invalid.yaml": "This is not valid yaml.",
			},
			expectedErrorCodes: []string{vet.UndefinedErrorCode},
		},
		{
			testName:    "A subdir of system is an error",
			root:        "foo",
			clusterName: "cluster-1",
			vet:         true,
			testFiles: fstesting.FileContentMap{
				"system/nomos.yaml":  aRepo,
				"system/sub/rb.yaml": aRepo,
			},
			expectedErrorCodes: []string{
				vet.IllegalSubdirectoryErrorCode,
				vet.MultipleRepoDefinitionsErrorCode,
				vet.MetadataNameCollisionErrorCode},
		},
		{
			testName:    "Objects in non-namespaces/ with an invalid label is an error",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				"system/nomos.yaml": `
kind: Repo
apiVersion: nomos.dev/v1alpha1
spec:
  version: "0.1.0"
metadata:
  name: repo
  labels:
    nomos.dev/illegal-label: "true"`,
			},
			expectedErrorCodes: []string{vet.IllegalLabelDefinitionErrorCode},
		},
		{
			testName:    "Objects in non-namespaces/ with an invalid annotation is an error",
			root:        "foo",
			clusterName: "cluster-1",
			testFiles: fstesting.FileContentMap{
				"system/nomos.yaml": `
kind: Repo
apiVersion: nomos.dev/v1alpha1
spec:
  version: "0.1.0"
metadata:
  name: repo
  annotations:
    nomos.dev/unsupported: "true"`,
			},
			expectedErrorCodes: []string{vet.IllegalAnnotationDefinitionErrorCode},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, test.Run)
	}
}

func TestEmptyDirectories(t *testing.T) {
	d := newTestDir(t, "foo")
	defer d.remove()

	// Create required repo definition.
	d.createTestFile(filepath.Join(repo.SystemDir, "nomos.yaml"), aRepo)

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
					Extension: ParserConfigFactory(),
				},
			)
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}

			actualPolicies, err := p.Parse(d.rootDir, "", time.Time{})
			if err != nil {
				t.Fatalf("unexpected error: %#v", err)
			}
			expectedPolicies := &v1.AllPolicies{
				PolicyNodes: map[string]v1.PolicyNode{
					v1.RootPolicyNodeName: createRootPN(nil),
				},
				ClusterPolicy: createClusterPolicy(),
				Syncs:         map[string]v1alpha1.Sync{},
			}
			if !cmp.Equal(actualPolicies, expectedPolicies) {
				t.Errorf("actual and expected AllPolicies didn't match: %v", cmp.Diff(actualPolicies, expectedPolicies))
			}
		})
	}
}
