package filesystem_test

import (
	"bytes"
	"encoding/json"
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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/backend"
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
	"github.com/google/nomos/testing/parsertest"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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
{{- end}}
{{- if .ResourceVersion}}
  resourceVersion: '{{.ResourceVersion}}'
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
{{template "objectmetatemplate" .}}
rules:
- apiGroups: ["batch/v1"]
  resources: ["jobs"]
  verbs:
   - "*"
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

	aReplicasetWithOwnerRefTemplate = `
kind: ReplicaSet
apiVersion: apps/v1
metadata:
  name: replicaset{{.ID}}
  ownerReferences:
  - apiVersion: apps/v1
    kind: Deployment
    name: some_deployment
    uid: some_uid
{{template "objectmetatemplate" .}}
spec:
  replicas: 1
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
  version: "1.0.0"
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

	anEngineerTemplate = `
apiVersion: employees/v1alpha1
kind: Engineer
metadata:
  name: philo
spec:
  cafePreference: 3
`

	anEngineerCRDTemplate = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: engineers.employees
spec:
  group: employees
  version: v1alpha1
  scope: Namespaced
  names:
    plural: engineers
    singular: engineer
    kind: Engineer
  versions:
    - name: v1alpha1
      served: true
      storage: true
`

	aNodeTemplate = `
apiVersion: v1
kind: Node
metadata:
  name: gke-1234
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
	aReplicaSetWithOwnerRef            = tpl("aReplicaSetWithOwnerRef", aReplicasetWithOwnerRefTemplate)
	aHierarchyConfig                   = tpl("aHierarchyConfig", aHierarchyConfigTemplate)
	anEngineer                         = tpl("anEngineer", anEngineerTemplate)
	anEngineerCRD                      = tpl("anEngineerCRD", anEngineerCRDTemplate)
	aNode                              = tpl("aNode", aNodeTemplate)
	aNamespaceSelector                 = tpl("aNamespaceSelectorTemplate", aNamespaceSelectorTemplate)
	aNamedRole                         = tpl("aNamedRole", aNamedRoleTemplate)
	anUndefinedResource                = tpl("AnUndefinedResource", anUndefinedResourceTemplate)
)

// templateData can be used to format any of the below values into templates to create
// a repository file set.
type templateData struct {
	ID, Name, Namespace, Attribute string
	Group, Version, Kind           string
	ResourceVersion                string
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
}

func (d testDir) remove() {
	os.RemoveAll(d.rootDir)
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

type Configs struct {
	RolesV1         []rbacv1.Role
	RoleBindingsV1  []rbacv1.RoleBinding
	ResourceQuotaV1 *corev1.ResourceQuota
	Resources       []v1.GenericResources
}

// createNamespaceConfig constructs a NamespaceConfig based on a Configs struct.
func createNamespaceConfig(name string, configs *Configs) v1.NamespaceConfig {
	pn := &v1.NamespaceConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceConfig",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NamespaceConfigSpec{},
	}
	if configs == nil {
		return *pn
	}

	if len(configs.RolesV1) > 0 {
		var roleObjects []runtime.Object
		for _, role := range configs.RolesV1 {
			roleObjects = append(roleObjects, runtime.Object(&role))
		}
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects(roleObjects, rbacv1.SchemeGroupVersion, "Role")...)
	}
	if len(configs.RoleBindingsV1) > 0 {
		var rbObjects []runtime.Object
		for _, rb := range configs.RoleBindingsV1 {
			rbObjects = append(rbObjects, runtime.Object(&rb))
		}
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects(rbObjects, rbacv1.SchemeGroupVersion, "RoleBinding")...)
	}
	if configs.ResourceQuotaV1 != nil {
		o := runtime.Object(configs.ResourceQuotaV1)
		pn.Spec.Resources = append(pn.Spec.Resources, resourcesFromObjects([]runtime.Object{o}, corev1.SchemeGroupVersion, "ResourceQuota")...)
	}
	if configs.Resources != nil {
		pn.Spec.Resources = append(pn.Spec.Resources, configs.Resources...)
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

func createNamespacePN(path string, configs *Configs) v1.NamespaceConfig {
	return createPNWithMeta(path, configs, nil, nil)
}

func createPNWithMeta(path string, configs *Configs, labels, annotations map[string]string) v1.NamespaceConfig {
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["configmanagement.gke.io/source-path"] = path
	pn := createNamespaceConfig(filepath.Base(path), configs)
	pn.Labels = labels
	pn.Annotations = annotations
	return pn
}

// createClusterConfigWithSpec creates a NamespaceConfig from the given spec and name.
func createClusterConfigWithSpec(name string, spec *v1.ClusterConfigSpec) *v1.ClusterConfig {
	return &v1.ClusterConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterConfig",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

func createClusterConfig() *v1.ClusterConfig {
	return createClusterConfigWithSpec(v1.ClusterConfigName, &v1.ClusterConfigSpec{})
}

func createCRDClusterConfig() *v1.ClusterConfig {
	return createClusterConfigWithSpec(v1.CRDClusterConfigName, &v1.ClusterConfigSpec{})
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
	testName                  string
	testFiles                 fstesting.FileContentMap
	vet                       bool
	expectedNamespaceConfigs  map[string]v1.NamespaceConfig
	expectedNumConfigs        map[string]int
	expectedClusterConfig     *v1.ClusterConfig
	expectedCRDClusterConfig  *v1.ClusterConfig
	expectedNumClusterConfigs *int
	expectedSyncs             map[string]v1.Sync
	expectedErrorCodes        []string
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
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Namespace dir with JSON Namespace",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.json": templateData{Name: "bar"}.apply(aNamespaceJSON),
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Namespace dir with Namespace with labels/annotations",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespaceWithLabelsAndAnnotations),
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createPNWithMeta("namespaces/bar", nil,
				map[string]string{"env": "prod"}, map[string]string{"audit": "true"}),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Namespace dir with ignored files",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Namespace dir with 2 ignored files",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
			"namespaces/bar/ignore2": "blah blah blah",
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Empty namespace dir",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/ignore":  "",
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar", nil),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
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
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{
					ResourceQuotaV1: createResourceQuota(
						"namespaces/bar/rq.yaml", "pod-quota", nil),
				},
			),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "ResourceQuota without declared HierarchyConfig",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{
					ResourceQuotaV1: createResourceQuota(
						"namespaces/bar/rq.yaml", "pod-quota", nil),
				},
			),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "ResourceQuota with scope and no hierarchical quota",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":         templateData{Kind: "ResourceQuota", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/rq.yaml": templateData{ID: "1", Scope: true, ScopeSelector: true}.apply(aQuota),
		},
		expectedNumConfigs: map[string]int{
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
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{ResourceQuotaV1: createResourceQuota(
					"namespaces/bar/combo.yaml", "pod-quota", nil),
				},
			),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "Namespace dir with multiple Roles",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/role1.yaml": templateData{ID: "1"}.apply(aRole),
			"namespaces/bar/role2.yaml": templateData{ID: "2"}.apply(aRole),
		},
		expectedNumConfigs: map[string]int{"bar": 2},
	},
	{
		testName: "Namespace dir with deployment",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":         templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/deployment.yaml": aDeploymentTemplate,
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{Resources: []v1.GenericResources{
					createDeployment(),
				},
				}),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            singleSyncMap("apps", "Deployment"),
	},
	{
		testName: "Namespace dir with Custom Resource",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/philo.yaml": templateData{ID: "1"}.apply(anEngineer),
		},
		expectedNumConfigs: map[string]int{"bar": 1},
	},
	{
		testName: "Cluster dir with CustomResourceDefinition",
		testFiles: fstesting.FileContentMap{
			"cluster/engineer-crd.yaml": templateData{}.apply(anEngineerCRD),
		},
		expectedClusterConfig: createClusterConfig(),
		expectedCRDClusterConfig: createClusterConfigWithSpec(
			v1.CRDClusterConfigName,
			&v1.ClusterConfigSpec{
				Resources: []v1.GenericResources{{
					Group: kinds.CustomResourceDefinition().Group,
					Kind:  kinds.CustomResourceDefinition().Kind,
					Versions: []v1.GenericVersionResources{{
						Version: "v1beta1",
						Objects: []runtime.RawExtension{{
							Object: &v1beta1.CustomResourceDefinition{
								TypeMeta: metav1.TypeMeta{
									Kind:       kinds.CustomResourceDefinition().Kind,
									APIVersion: v1beta1.SchemeGroupVersion.String(),
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "engineers.employees",
									Annotations: map[string]string{
										v1.SourcePathAnnotationKey: "cluster/engineer-crd.yaml",
									},
								},
								Spec: v1beta1.CustomResourceDefinitionSpec{
									Group: "employees",
									Scope: v1beta1.NamespaceScoped,
									Names: v1beta1.CustomResourceDefinitionNames{
										Plural:   "engineers",
										Singular: "engineer",
										Kind:     "Engineer",
									},
									Version: "v1alpha1",
									Versions: []v1beta1.CustomResourceDefinitionVersion{{
										Name:    "v1alpha1",
										Served:  true,
										Storage: true,
									}},
								},
							},
						}},
					}},
				}},
			}),
		expectedSyncs: singleSyncMap(kinds.CustomResourceDefinition().Group, kinds.CustomResourceDefinition().Kind),
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
		expectedNumConfigs: map[string]int{"bar": 2},
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
		expectedErrorCodes: []string{vet.UnsyncableResourcesErrorCode},
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
		testName: "Abstract Namespace dir with Namespace and Abstract namespace children is valid",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/role.yaml":      templateData{Name: "role"}.apply(aRole),
			"namespaces/bar/baz/nssel.yaml": templateData{Name: "dummy"}.apply(aNamespaceSelector),
			"namespaces/bar/qux/ns.yaml":    templateData{Name: "qux"}.apply(aNamespace),
		},
	},
	{
		testName: "Namespace dir with Abstract Namespace child",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns.yaml":       templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/bar/baz/role.yaml": templateData{Name: "role"}.apply(aRole),
		},
		expectedErrorCodes: []string{vet.IllegalNamespaceSubdirectoryErrorCode, vet.UnsyncableResourcesErrorCode},
	},
	{
		testName: "Abstract Namespace dir with ignored file",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ignore": "",
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Abstract Namespace dir with RoleBinding and no descendants, default inheritance",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedNumConfigs: map[string]int{},
		expectedErrorCodes: []string{vet.UnsyncableResourcesErrorCode},
	},
	{
		testName: "Abstract Namespace dir with RoleBinding, hierarchicalQuota mode specified",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":             templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", HierarchyMode: "hierarchicalQuota"}.apply(aHierarchyConfig),
			"namespaces/bar/rb1.yaml":    templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalHierarchyModeErrorCode},
	},
	{
		testName: "Abstract Namespace dir with RoleBinding, inheritance off",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":             templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/rb1.yaml":    templateData{ID: "1"}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
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
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{RoleBindingsV1: rbs(templateData{Annotations: map[string]string{
					v1.SourcePathAnnotationKey: "namespaces/rb.yaml",
				}})}),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            singleSyncMap(kinds.RoleBinding().Group, kinds.RoleBinding().Kind),
	},
	{
		testName: "Namespaces dir with ResourceQuota, default inheritance",
		testFiles: fstesting.FileContentMap{
			"namespaces/rq.yaml":     templateData{}.apply(aQuota),
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{ResourceQuotaV1: createResourceQuota("namespaces/rq.yaml", "pod-quota", nil)}),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "Abstract Namespace dir with uninheritable ResourceQuota, inheritance off",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":             templateData{Kind: "ResourceQuota", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/rq.yaml":     templateData{}.apply(aQuota),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Abstract Namespace dir with ResourceQuota, inherit specified",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":             templateData{Kind: "ResourceQuota", HierarchyMode: "inherit"}.apply(aHierarchyConfig),
			"namespaces/bar/rq.yaml":     templateData{}.apply(aQuota),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"baz": createNamespacePN("namespaces/bar/baz",
				&Configs{ResourceQuotaV1: createResourceQuota("namespaces/bar/rq.yaml", "pod-quota", nil)}),
		},
		expectedSyncs: singleSyncMap("", "ResourceQuota"),
	},
	{
		testName: "Abstract Namespace dir with uninheritable Rolebinding",
		testFiles: fstesting.FileContentMap{
			"system/rb.yaml":             templateData{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", HierarchyMode: "none"}.apply(aHierarchyConfig),
			"namespaces/bar/rb.yaml":     templateData{}.apply(aRoleBinding),
			"namespaces/bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalAbstractNamespaceObjectKindErrorCode},
	},
	{
		testName: "Abstract Namespace dir with NamespaceSelector CRD",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns-selector.yaml": templateData{}.apply(aNamespaceSelector),
			"namespaces/bar/baz/ns.yaml":      templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedNumConfigs: map[string]int{},
	},
	{
		testName: "Abstract Namespace dir with NamespaceSelector CRD and object",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/ns-selector.yaml": templateData{}.apply(aNamespaceSelector),
			"namespaces/bar/rb.yaml":          templateData{ID: "1", LBPName: "sre-supported"}.apply(aLBPRoleBinding),
			"namespaces/bar/prod-ns/ns.yaml":  templateData{Name: "prod-ns", Labels: map[string]string{"environment": "prod"}}.apply(aNamespace),
			"namespaces/bar/test-ns/ns.yaml":  templateData{Name: "test-ns"}.apply(aNamespace),
		},
		expectedNumConfigs: map[string]int{"prod-ns": 1, "test-ns": 0},
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
		testName:                 "Minimal repo",
		testFiles:                fstesting.FileContentMap{},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "Only system dir with valid HierarchyConfig",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml": templateData{Kind: "ResourceQuota"}.apply(aHierarchyConfig),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            syncMap(),
	},
	{
		testName: "Multiple resources with HierarchyConfigs",
		testFiles: fstesting.FileContentMap{
			"system/rq.yaml":   templateData{Kind: "ResourceQuota"}.apply(aHierarchyConfig),
			"system/role.yaml": templateData{Group: "rbac.authorization.k8s.io", Kind: "Role"}.apply(aHierarchyConfig),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            syncMap(),
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
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs:            syncMap(),
	},
	{
		testName: "Namespaces dir with ignored file",
		testFiles: fstesting.FileContentMap{
			"namespaces/ignore": "",
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
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
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"bar": createNamespacePN("namespaces/bar",
				&Configs{ResourceQuotaV1: createResourceQuota(
					"namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewConfigManagementQuotaLabels()),
				}),
		},
		expectedClusterConfig: createClusterConfigWithSpec(
			v1.ClusterConfigName,
			&v1.ClusterConfigSpec{
				Resources: []v1.GenericResources{
					{
						Group: configmanagement.GroupName,
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
																	"namespaces/rq.yaml", resourcequota.ResourceQuotaObjectName, resourcequota.NewConfigManagementQuotaLabels()),
															},
														},
													},
												},
											}),
									},
								},
							}}}}}),
		expectedCRDClusterConfig: createCRDClusterConfig(),
		expectedSyncs: syncMap(
			makeSync(configmanagement.GroupName, "HierarchicalQuota"),
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
		expectedNumConfigs: map[string]int{"bar": 2},
	},
	{
		testName: "Cluster dir with multiple ClusterRoles",
		testFiles: fstesting.FileContentMap{
			"cluster/cr1.yaml": templateData{ID: "1"}.apply(aClusterRole),
			"cluster/cr2.yaml": templateData{ID: "2"}.apply(aClusterRole),
		},
		expectedNumClusterConfigs: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple ClusterRoleBindings",
		testFiles: fstesting.FileContentMap{
			"cluster/crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"cluster/crb2.yaml": templateData{ID: "2"}.apply(aClusterRoleBinding),
		},
		expectedNumClusterConfigs: toIntPointer(2),
	},
	{
		testName: "Cluster dir with multiple PodSecurityPolicies",
		testFiles: fstesting.FileContentMap{
			"cluster/psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"cluster/psp2.yaml": templateData{ID: "2"}.apply(aPodSecurityPolicy),
		},
		expectedNumClusterConfigs: toIntPointer(2),
	},
	{
		testName: "Cluster dir with deployment",
		testFiles: fstesting.FileContentMap{
			"cluster/node.yaml": templateData{}.apply(aNode),
		},
		expectedNumClusterConfigs: toIntPointer(1),
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
		testName: "Two abstract Namespace dirs with non-unique names are allowed.",
		testFiles: fstesting.FileContentMap{
			"namespaces/bar/baz/corge/ns.yaml": templateData{Name: "corge"}.apply(aNamespace),
			"namespaces/qux/baz/waldo/ns.yaml": templateData{Name: "waldo"}.apply(aNamespace),
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"corge": createNamespacePN("namespaces/bar/baz/corge",
				&Configs{}),
			"waldo": createNamespacePN("namespaces/qux/baz/waldo",
				&Configs{}),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "An abstract namespace and a leaf namespace may share a name",
		testFiles: fstesting.FileContentMap{
			"namespaces/waldo/corge/ns.yaml": templateData{Name: "corge"}.apply(aNamespace),
			"namespaces/bar/waldo/ns.yaml":   templateData{Name: "waldo"}.apply(aNamespace),
		},
		expectedNamespaceConfigs: map[string]v1.NamespaceConfig{
			"corge": createNamespacePN("namespaces/waldo/corge",
				&Configs{}),
			"waldo": createNamespacePN("namespaces/bar/waldo",
				&Configs{}),
		},
		expectedClusterConfig:    createClusterConfig(),
		expectedCRDClusterConfig: createCRDClusterConfig(),
	},
	{
		testName: "kube-* is a system dir but is allowed",
		testFiles: fstesting.FileContentMap{
			"namespaces/kube-something/ns.yaml": templateData{Name: "kube-something"}.apply(aNamespace),
		},
	},
	{
		testName: "kube-system is a system dir but is allowed",
		testFiles: fstesting.FileContentMap{
			"namespaces/kube-system/ns.yaml": templateData{Name: "kube-system"}.apply(aNamespace),
		},
	},
	{
		testName: "Default namespace is allowed",
		testFiles: fstesting.FileContentMap{
			"namespaces/default/ns.yaml": templateData{Name: "default"}.apply(aNamespace),
		},
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
		testName: "NamespaceSelector may not have clusterSelector annotations",
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
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
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
			"namespaces/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
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
		expectedNumConfigs: map[string]int{
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
			"namespaces/fox/bar/ns.yaml":   templateData{Name: "bar"}.apply(aNamespace),
			"namespaces/fox/qux/rb-2.yaml": templateData{ID: "alice"}.apply(aRoleBinding),
			"namespaces/fox/qux/ns.yaml":   templateData{Name: "qux"}.apply(aNamespace),
		},
	},
	{
		testName: "Empty string name is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/bar/role1.yaml": templateData{Name: ""}.apply(aNamedRole),
			"namespaces/foo/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.MissingObjectNameErrorCode},
	},
	{
		testName: "No name is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/bar/role1.yaml": templateData{}.apply(aNamedRole),
			"namespaces/foo/bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.MissingObjectNameErrorCode},
	},
	{
		testName: "Specifying system generated field is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/bar/role.yaml": templateData{ResourceVersion: "999"}.apply(aRole),
			"namespaces/foo/bar/ns.yaml":   templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalFieldsInConfigErrorCode},
	},
	{
		testName: "Repo outside system/ is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/repo.yaml": aRepo,
			"namespaces/foo/ns.yaml":   templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalSystemResourcePlacementErrorCode},
	},
	{
		testName: "HierarchyConfig outside system/ is an error",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/config.yaml": templateData{Group: "rbac.authorization.k8s.io", Kind: "Role"}.apply(aHierarchyConfig),
			"namespaces/foo/ns.yaml":     templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedErrorCodes: []string{vet.IllegalSystemResourcePlacementErrorCode},
	},
	{
		testName: "HierarchyConfig contains a CRD",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: kinds.CustomResourceDefinition().Group, Kind: kinds.CustomResourceDefinition().Kind}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{
			vet.ClusterScopedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "HierarchyConfig contains a Namespace",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Kind: "Namespace"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{vet.UnsupportedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "HierarchyConfig contains a NamespaceConfig",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: configmanagement.GroupName, Kind: "NamespaceConfig"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{
			vet.UnsupportedResourceInHierarchyConfigErrorCode,
			vet.ClusterScopedResourceInHierarchyConfigErrorCode},
	},
	{
		testName: "HierarchyConfig contains a Sync",
		testFiles: fstesting.FileContentMap{
			"system/config.yaml": templateData{Group: configmanagement.GroupName, Kind: "Sync"}.apply(aHierarchyConfig),
		},
		expectedErrorCodes: []string{
			vet.UnsupportedResourceInHierarchyConfigErrorCode,
			vet.ClusterScopedResourceInHierarchyConfigErrorCode},
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
	{
		testName: "ReplicaSet with OwnerReferences specified.",
		testFiles: fstesting.FileContentMap{
			"namespaces/foo/namespace.yaml":  templateData{Name: "foo"}.apply(aNamespace),
			"namespaces/foo/replicaset.yaml": templateData{}.apply(aReplicaSetWithOwnerRef),
		},
		expectedErrorCodes: []string{vet.IllegalFieldsInConfigErrorCode},
	},
}

func (tc *parserTestCase) Run(t *testing.T) {
	d := newTestDir(t)
	defer d.remove()

	if glog.V(6) {
		glog.Infof("Testcase: %+v", spew.Sdump(tc))
	}

	for k, v := range tc.testFiles {
		// stuff
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
			Vet:       tc.vet,
			Validate:  true,
			Extension: &filesystem.NomosVisitorProvider{},
			RootPath:  rootPath,
		},
	)
	actualConfigs, mErr := p.Parse("", &namespaceconfig.AllConfigs{}, time.Time{}, tc.clusterName)

	vettesting.ExpectErrors(tc.expectedErrorCodes, mErr, t)
	if mErr != nil {
		// We expected there to be an error, so no need to do config validation
		return
	}

	if actualConfigs == nil {
		t.Fatalf("actualConfigs is nil")
	}

	if len(tc.expectedNumConfigs) > 0 {
		n := make(map[string]int)
		for k, v := range actualConfigs.NamespaceConfigs {
			n[k] = 0
			for _, res := range v.Spec.Resources {
				for _, version := range res.Versions {
					n[k] += len(version.Objects)
				}
			}
		}
		if !cmp.Equal(n, tc.expectedNumConfigs) {
			t.Errorf("Actual and expected number of configs didn't match: %v", cmp.Diff(n, tc.expectedNumConfigs))
		}
	}

	if tc.expectedNumClusterConfigs != nil {
		actualNumClusterConfigs := 0
		for _, res := range actualConfigs.ClusterConfig.Spec.Resources {
			for _, version := range res.Versions {
				actualNumClusterConfigs += len(version.Objects)
			}
		}

		if !cmp.Equal(actualNumClusterConfigs, *tc.expectedNumClusterConfigs) {
			t.Errorf("Actual and expected number of cluster configs didn't match: %v", cmp.Diff(actualNumClusterConfigs,
				tc.expectedNumClusterConfigs))
		}
	}

	if tc.expectedNamespaceConfigs != nil || tc.expectedClusterConfig != nil || tc.expectedCRDClusterConfig != nil ||
		tc.expectedSyncs != nil {
		if tc.expectedNamespaceConfigs == nil {
			tc.expectedNamespaceConfigs = map[string]v1.NamespaceConfig{}
		}
		if tc.expectedSyncs == nil {
			tc.expectedSyncs = map[string]v1.Sync{}
		}

		expectedConfigs := &namespaceconfig.AllConfigs{
			NamespaceConfigs: tc.expectedNamespaceConfigs,
			ClusterConfig:    tc.expectedClusterConfig,
			CRDClusterConfig: tc.expectedCRDClusterConfig,
			Syncs:            tc.expectedSyncs,
			Repo:             fake.RepoObject(),
		}
		if d := cmp.Diff(expectedConfigs, actualConfigs, resourcequota.ResourceQuantityEqual(), cmpopts.EquateEmpty()); d != "" {
			t.Errorf("Actual and expected configs didn't match: diff\n%v", d)
		}
	}
}

func TestParser(t *testing.T) {
	for _, tc := range parserTestCases {
		tc.testFiles["system/repo.yaml"] = aRepo
		t.Run(tc.testName, tc.Run)
	}
}

func clusterSelectorAnnotation(value string) object.MetaMutator {
	return object.Annotation(v1.ClusterSelectorAnnotationKey, value)
}

func inlinedClusterSelectorAnnotation(t *testing.T, selector *v1.ClusterSelector) object.MetaMutator {
	content, err := json.Marshal(selector)
	if err != nil {
		t.Error(err)
	}
	return object.Annotation(v1.ClusterSelectorAnnotationKey, string(content))
}

func cluster(name string, opts ...object.MetaMutator) ast.FileObject {
	mutators := append(opts, object.Name(name))
	return fake.Cluster(mutators...)
}

func clusterSelectorObject(name, key, value string) *v1.ClusterSelector {
	obj := fake.ClusterSelectorObject(object.Name(name))
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func inlinedSelectorAnnotation(t *testing.T, selector *v1.ClusterSelector) object.MetaMutator {
	return inlinedClusterSelectorAnnotation(t, selector)
}

func inCluster(clusterName string) object.MetaMutator {
	return object.Annotation(v1.ClusterNameAnnotationKey, clusterName)
}

func source(path string) object.MetaMutator {
	return object.Annotation(v1.SourcePathAnnotationKey, path)
}

// clusterConfig generates a valid ClusterConfig to be put in AllConfigs given the set of hydrated
// cluster-scoped runtime.Objects.
func clusterConfig(objects ...runtime.Object) *v1.ClusterConfig {
	config := fake.ClusterConfigObject(fake.ClusterConfigMeta())
	config.Spec.Resources = genericResources(objects...)
	return config
}

// namespaceConfig generates a valid NamespaceConfig to be put in AllConfigs given the set of
// hydrated runtime.Objects for that Namespace.
func namespaceConfig(clusterName, dir string, opt object.MetaMutator, objects ...runtime.Object) v1.NamespaceConfig {
	config := fake.NamespaceConfigObject(fake.NamespaceConfigMeta(inCluster(clusterName), source(dir)))
	config.Name = cmpath.FromSlash(dir).Base()
	config.Spec.Resources = genericResources(objects...)
	if opt != nil {
		opt(config)
	}
	return *config
}

// namespaceConfigs turns a list of NamespaceConfigs into the map AllConfigs requires.
func namespaceConfigs(ncs ...v1.NamespaceConfig) map[string]v1.NamespaceConfig {
	result := map[string]v1.NamespaceConfig{}
	for _, nc := range ncs {
		result[nc.Name] = nc
	}
	return result
}

// genericResources convers a list of runtime.Objects to the GenericResources array required for
// AllConfigs.
func genericResources(objects ...runtime.Object) []v1.GenericResources {
	var result []v1.GenericResources
	for _, obj := range objects {
		result = backend.AppendResource(result, obj)
	}
	return result
}

// syncs generates the sync map to be put in AllConfigs.
func syncs(gvks ...schema.GroupVersionKind) map[string]v1.Sync {
	result := map[string]v1.Sync{}
	for _, gvk := range gvks {
		result[groupKind(gvk)] = *fake.SyncObject(gvk.GroupKind())
	}
	return result
}

// groupKind factors out the two-line operation of getting the GroupKind string from a
// GroupVersionKind. The GroupKind.String() method has a pointer receiver, so
// gvk.GroupKind.String() is an error.
func groupKind(gvk schema.GroupVersionKind) string {
	gk := gvk.GroupKind()
	return strings.ToLower(gk.String())
}

// Test how the parser handles ClusterSelectors
func TestParseClusterSelector(t *testing.T) {
	prodCluster := "cluster-1"
	devCluster := "cluster-2"

	prodSelectorName := "sel-1"
	prodLabel := object.Label("environment", "prod")
	prodSelectorObject := func() *v1.ClusterSelector {
		return clusterSelectorObject(prodSelectorName, "environment", "prod")
	}
	prodSelectorAnnotation := clusterSelectorAnnotation(prodSelectorName)
	prodSelectorAnnotationInlined := inlinedSelectorAnnotation(t, prodSelectorObject())

	devSelectorName := "sel-2"
	devLabel := object.Label("environment", "dev")
	devSelectorObject := func() *v1.ClusterSelector {
		return clusterSelectorObject(devSelectorName, "environment", "dev")
	}
	devSelectorAnnotation := clusterSelectorAnnotation(devSelectorName)
	devSelectorAnnotationInlined := inlinedSelectorAnnotation(t, devSelectorObject())

	test := parsertest.VetTest(
		parsertest.Success("Resource without selector always exists 1",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(inCluster(prodCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Resource without selector always exists 2",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(devCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(inCluster(devCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Namespace resource selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(prodSelectorAnnotationInlined, inCluster(prodCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace resource not selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(devCluster, "namespaces/bar", nil),
				),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Success("Namespace selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/bar", prodSelectorAnnotationInlined,
						fake.RoleBindingObject(inCluster(prodCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace not selected",
			nil,
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Cluster resource selected",
			&namespaceconfig.AllConfigs{
				ClusterConfig: clusterConfig(
					fake.ClusterRoleBindingObject(prodSelectorAnnotationInlined, inCluster(prodCluster), source("cluster/crb.yaml")),
				),
				Syncs: syncs(kinds.ClusterRoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.ClusterRoleBinding(prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Cluster resource not selected",
			nil,
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.ClusterRoleBinding(prodSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Success("Abstract Namespace resouce selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/foo/bar", nil,
						fake.ConfigMapObject(prodSelectorAnnotationInlined, inCluster(prodCluster), source("namespaces/foo/configmap.yaml")),
					),
				),
				Syncs: syncs(kinds.ConfigMap()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.ConfigMap(), v1.HierarchyModeInherit)),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/foo/bar"),
			fake.ConfigMapAtPath("namespaces/foo/configmap.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Colliding resources selected to different clusters may coexist",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(devCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(devSelectorAnnotationInlined, inCluster(devCluster), source("namespaces/bar/rolebinding-2.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),
			fake.FileObject(devSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding-1.yaml", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding-2.yaml", devSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Failure(
			"A namespaced object that has a cluster selector annotation for nonexistent cluster is an error",
			vet.ObjectHasUnknownClusterSelectorCode,
			fake.Namespace("namespaces/foo", clusterSelectorAnnotation("does-not-exist")),
		),
		parsertest.Failure(
			"A cluster object that has a cluster selector annotation for nonexistent cluster is an error",
			vet.ObjectHasUnknownClusterSelectorCode,
			fake.ClusterRole(clusterSelectorAnnotation("does-not-exist")),
		))

	test.RunAll(t)
}

func TestParserVet(t *testing.T) {
	test := parsertest.VetTest(
		parsertest.Failure("A subdir of system is an error",
			vet.IllegalSubdirectoryErrorCode,
			fake.HierarchyConfigAtPath("system/sub/hc.yaml")),
		parsertest.Failure("Objects in non-namespaces/ with an invalid label is an error",
			vet.IllegalLabelDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				fake.HierarchyConfigMeta(object.Label("configmanagement.gke.io/illegal-label", "true"))),
		),
		parsertest.Failure("Objects in non-namespaces/ with an invalid annotation is an error",
			vet.IllegalAnnotationDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				fake.HierarchyConfigMeta(object.Annotation("configmanagement.gke.io/illegal-annotation", "true"))),
		))

	test.RunAll(t)
}
