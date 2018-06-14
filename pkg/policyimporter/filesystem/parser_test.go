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
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	aNamespaceTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Name}}
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
	numPolicies         = 2
)

type templateData struct {
	ID, Name, Namespace, Attribute string
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
	RolesV1         []rbac_v1.Role
	RoleBindingsV1  []rbac_v1.RoleBinding
	ResourceQuotaV1 *core_v1.ResourceQuota
}

func createPolicyNode(
	name string,
	parent string,
	nodeType policyhierarchy_v1.PolicyNodeType,
	policies *Policies) policyhierarchy_v1.PolicyNode {
	pn := policynode.NewPolicyNode(name,
		&policyhierarchy_v1.PolicyNodeSpec{
			Type:   nodeType,
			Parent: parent,
		})
	if policies != nil {
		pn.Spec.RolesV1 = policies.RolesV1
		pn.Spec.RoleBindingsV1 = policies.RoleBindingsV1
		pn.Spec.ResourceQuotaV1 = policies.ResourceQuotaV1
	}
	return *pn
}

func createNamespacePN(
	name string,
	parent string,
	policies *Policies) policyhierarchy_v1.PolicyNode {
	return createPolicyNode(name, parent, policyhierarchy_v1.Namespace, policies)
}

func createPolicyspacePN(
	name string,
	parent string,
	policies *Policies) policyhierarchy_v1.PolicyNode {
	return createPolicyNode(name, parent, policyhierarchy_v1.Policyspace, policies)
}

func createReservedPN(
	name string,
	parent string,
	policies *Policies) policyhierarchy_v1.PolicyNode {
	return createPolicyNode(name, parent, policyhierarchy_v1.ReservedNamespace, policies)
}

func createClusterPolicy() *policyhierarchy_v1.ClusterPolicy {
	return policynode.NewClusterPolicy(policyhierarchy_v1.ClusterPolicyName,
		&policyhierarchy_v1.ClusterPolicySpec{})
}

func createResourceQuota(name string, namespace string) *core_v1.ResourceQuota {
	return &core_v1.ResourceQuota{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: core_v1.ResourceQuotaSpec{
			Hard: core_v1.ResourceList{"pods": resource.MustParse("10")},
		},
	}
}

type fileContentMap map[string]string

type parserTestCase struct {
	testName                   string
	root                       string
	testFiles                  fileContentMap
	expectedPolicyNodes        map[string]policyhierarchy_v1.PolicyNode
	expectedNumPolicies        map[string]int
	expectedClusterPolicy      *policyhierarchy_v1.ClusterPolicy
	expectedNumClusterPolicies *int
	expectedError              bool
}

var parserTestCases = []parserTestCase{
	{
		testName: "Namespace dir with YAML Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createNamespacePN("bar", "foo", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with JSON Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.json": templateData{Name: "bar"}.apply(aNamespaceJSON),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createNamespacePN("bar", "foo", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with ignored files",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/ignore":  "",
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createNamespacePN("bar", "foo", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with 2 ignored files",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/ignore":  "",
			"bar/ignore2": "blah blah blah",
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createNamespacePN("bar", "foo", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with multiple Namespaces",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"bar/ns2.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir without Namespace multiple",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ignore":  "",
			"bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with namespace mismatch",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with invalid name",
		root:     "foo",
		testFiles: fileContentMap{
			"baR/ns.yaml": templateData{Name: "baR"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with single ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/rq.yaml": templateData{Namespace: "bar"}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createNamespacePN("bar", "foo",
				&Policies{ResourceQuotaV1: createResourceQuota("pod-quota", "bar")}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with single ResourceQuota single file",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/combo.yaml": templateData{Name: "bar"}.apply(aNamespace) + "\n---\n" + templateData{Namespace: "bar"}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createNamespacePN("bar", "foo",
				&Policies{ResourceQuotaV1: createResourceQuota("pod-quota", "bar")}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Namespace dir with multiple ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml":  templateData{Name: "bar"}.apply(aNamespace),
			"bar/rq.yaml":  templateData{ID: "1", Namespace: "bar"}.apply(aQuota),
			"bar/rq2.yaml": templateData{ID: "2", Namespace: "bar"}.apply(aQuota),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ResourceQuota namespace mismatch",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/rq.yaml": templateData{Namespace: "baz"}.apply(aQuota),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with multiple Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"bar/role1.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aRole),
			"bar/role2.yaml": templateData{ID: "2", Namespace: "bar"}.apply(aRole),
		},
		expectedNumPolicies: map[string]int{"foo": 0, "bar": 2},
	},
	{
		testName: "Namespace dir with duplicate Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml":    templateData{Name: "bar"}.apply(aNamespace),
			"bar/role1.yaml": templateData{Namespace: "bar"}.apply(aRole),
			"bar/role2.yaml": templateData{Namespace: "bar"}.apply(aRole),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/r1.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"bar/r2.yaml": templateData{ID: "2", Namespace: "bar"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"foo": 0, "bar": 2},
	},
	{
		testName: "Namespace dir with duplicate Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/r1.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"bar/r2.yaml": templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with duplicate Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml":     templateData{Name: "bar"}.apply(aNamespace),
			"bar/r1.yaml":     templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"bar/r2.yaml":     templateData{ID: "1", Namespace: "bar"}.apply(aRoleBinding),
			"bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with non-conflicting reserved Namespace specified",
		root:     "foo",
		testFiles: fileContentMap{
			"reserved.yaml": templateData{Namespace: "baz", Attribute: string(policyhierarchy_v1.ReservedAttribute), Name: policyhierarchy_v1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
			"bar/ns.yaml":   templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedClusterPolicy: createClusterPolicy(),
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"baz": createReservedPN("baz", "", nil),
			"bar": createNamespacePN("bar", "foo", nil),
		},
	},
	{
		testName: "Namespace dir with non-conflicting reserved Namespace, but invalid attribute specified",
		root:     "foo",
		testFiles: fileContentMap{
			"reserved.yaml": templateData{Namespace: "foo", Attribute: "invalid-attribute", Name: policyhierarchy_v1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
			"bar/ns.yaml":   templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with conflicting reserved Namespace specified",
		root:     "foo",
		testFiles: fileContentMap{
			"reserved.yaml": templateData{Namespace: "foo", Attribute: "reserved", Name: policyhierarchy_v1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
			"foo/ns.yaml":   templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "reserved namespace ConfigMap with invalid name",
		root:     "foo",
		testFiles: fileContentMap{
			"reserved.yaml": templateData{Namespace: "foo", Attribute: "reserved", Name: "random-name"}.apply(aConfigMap),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ClusterRole",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/cr.yaml": templateData{}.apply(aClusterRole),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ClusterRoleBinding",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with PodSecurityPolicy",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with policyspace child",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml":    templateData{Name: "baz"}.apply(aNamespace),
			"bar/baz/ignore": "",
		},
		expectedError: true,
	},
	{
		testName: "Namespace dir with ConfigMap",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"bar/cm.yaml": templateData{Namespace: "foo", Attribute: "reserved", Name: policyhierarchy_v1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with ignored file",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/ignore": "",
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createPolicyspacePN("bar", "foo", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Policyspace dir with ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", nil),
			"bar": createPolicyspacePN("bar", "foo",
				&Policies{ResourceQuotaV1: createResourceQuota("pod-quota", "")}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Policyspace dir with ResourceQuota namespace set",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/rq.yaml": templateData{Namespace: "qux"}.apply(aQuota),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/rq.yaml":     templateData{Namespace: "bar"}.apply(aQuota),
			"bar/baz/ns.yaml": templateData{Name: "baz"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/role.yaml": templateData{}.apply(aRole),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"bar/rb2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"foo": 0, "bar": 2},
	},
	{
		testName: "Policyspace dir with ClusterRole",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/cr.yaml": templateData{}.apply(aClusterRole),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with ClusterRoleBinding",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/crb.yaml": templateData{}.apply(aClusterRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with PodSecurityPolicy",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/psp.yaml": templateData{}.apply(aPodSecurityPolicy),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace dir with ConfigMap",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/cm.yaml": templateData{Namespace: "foo", Attribute: "reserved", Name: policyhierarchy_v1.ReservedNamespacesConfigMapName}.apply(aConfigMap),
		},
		expectedError: true,
	},
	{
		testName: "Policyspace and Namespace dir have duplicate rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"bar/rb1.yaml":     templateData{ID: "1"}.apply(aRoleBinding),
			"bar/baz/ns.yaml":  templateData{Name: "baz"}.apply(aNamespace),
			"bar/baz/rb1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Root dir empty",
		root:     "foo",
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", &Policies{}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Root dir with ignore file",
		root:     "foo",
		testFiles: fileContentMap{
			"ignore": "",
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "", &Policies{}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Root dir with Namespace",
		root:     "foo",
		testFiles: fileContentMap{
			"ns.yaml": templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Root dir with ResourceQuota",
		root:     "foo",
		testFiles: fileContentMap{
			"rq.yaml": templateData{}.apply(aQuota),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyspacePN("foo", "",
				&Policies{ResourceQuotaV1: createResourceQuota("pod-quota", "")}),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Root dir with ResourceQuota and namespace dir",
		root:     "foo",
		testFiles: fileContentMap{
			"rq.yaml":     templateData{}.apply(aQuota),
			"bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedPolicyNodes: map[string]policyhierarchy_v1.PolicyNode{
			"foo": createPolicyNode("foo", "", policyhierarchy_v1.Policyspace,
				&Policies{ResourceQuotaV1: createResourceQuota("pod-quota", "")}),
			"bar": createNamespacePN("bar", "foo", nil),
		},
		expectedClusterPolicy: createClusterPolicy(),
	},
	{
		testName: "Root dir with Roles",
		root:     "foo",
		testFiles: fileContentMap{
			"role.yaml": templateData{}.apply(aRole),
		},
		expectedError: true,
	},
	{
		testName: "Root dir with multiple Rolebindings",
		root:     "foo",
		testFiles: fileContentMap{
			"r1.yaml": templateData{ID: "1"}.apply(aRoleBinding),
			"r2.yaml": templateData{ID: "2"}.apply(aRoleBinding),
		},
		expectedNumPolicies: map[string]int{"foo": 2},
	},
	{
		testName: "Root dir with multiple ClusterRoles",
		root:     "foo",
		testFiles: fileContentMap{
			"cr1.yaml": templateData{ID: "1"}.apply(aClusterRole),
			"cr2.yaml": templateData{ID: "2"}.apply(aClusterRole),
		},
		expectedNumClusterPolicies: &numPolicies,
	},
	{
		testName: "Root dir with multiple ClusterRoleBindings",
		root:     "foo",
		testFiles: fileContentMap{
			"crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"crb2.yaml": templateData{ID: "2"}.apply(aClusterRoleBinding),
		},
		expectedNumClusterPolicies: &numPolicies,
	},
	{
		testName: "Root dir with multiple PodSecurityPolicies",
		root:     "foo",
		testFiles: fileContentMap{
			"psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"psp2.yaml": templateData{ID: "2"}.apply(aPodSecurityPolicy),
		},
		expectedNumClusterPolicies: &numPolicies,
	},
	{
		testName: "Root dir with duplicate ClusterRole names",
		root:     "foo",
		testFiles: fileContentMap{
			"cr1.yaml": templateData{ID: "1"}.apply(aClusterRole),
			"cr2.yaml": templateData{ID: "1"}.apply(aClusterRole),
		},
		expectedError: true,
	},
	{
		testName: "Root dir with duplicate ClusterRoleBinding names",
		root:     "foo",
		testFiles: fileContentMap{
			"crb1.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
			"crb2.yaml": templateData{ID: "1"}.apply(aClusterRoleBinding),
		},
		expectedError: true,
	},
	{
		testName: "Root dir with duplicate PodSecurityPolicy names",
		root:     "foo",
		testFiles: fileContentMap{
			"psp1.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
			"psp2.yaml": templateData{ID: "1"}.apply(aPodSecurityPolicy),
		},
		expectedError: true,
	},
	{
		testName: "Dir name not unique 1",
		root:     "foo",
		testFiles: fileContentMap{
			"baz/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
			"qux/bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name not unique 2",
		root:     "foo",
		testFiles: fileContentMap{
			"foo/ns.yaml": templateData{Name: "foo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name not unique 3",
		root:     "foo",
		testFiles: fileContentMap{
			// Two policyspace dirs with same name.
			"bar/baz/corge/ns.yaml": templateData{Name: "corge"}.apply(aNamespace),
			"qux/baz/waldo/ns.yaml": templateData{Name: "waldo"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name reserved",
		root:     "foo",
		testFiles: fileContentMap{
			"kube-system/ns.yaml": templateData{Name: "kube-system"}.apply(aNamespace),
		},
		expectedError: true,
	},
	{
		testName: "Dir name invalid",
		root:     "foo",
		testFiles: fileContentMap{
			"foo bar/ns.yaml": templateData{Name: "bar"}.apply(aNamespace),
		},
		expectedError: true,
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

			p, err := NewParser(false)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			actualPolicies, err := p.Parse(d.rootDir)
			if tc.expectedError {
				if err != nil {
					return
				}
				t.Fatalf("Expected error")
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(tc.expectedNumPolicies) > 0 {
				n := make(map[string]int)
				for k, v := range actualPolicies.PolicyNodes {
					p := v.Spec
					n[k] = len(p.RolesV1) + len(p.RoleBindingsV1)
					if p.ResourceQuotaV1 != nil {
						n[k]++
					}
				}
				if diff := deep.Equal(n, tc.expectedNumPolicies); diff != nil {
					t.Fatalf("Actual and expected number of policy nodes didn't match: %v", diff)
				}
			}

			if tc.expectedNumClusterPolicies != nil {
				p := actualPolicies.ClusterPolicy.Spec
				n := len(p.ClusterRolesV1) + len(p.ClusterRoleBindingsV1) + len(p.PodSecurityPoliciesV1Beta1)
				if diff := deep.Equal(n, *tc.expectedNumClusterPolicies); diff != nil {
					t.Fatalf("Actual and expected number of cluster policies didn't match: %v", diff)
				}
			}

			if tc.expectedPolicyNodes != nil || tc.expectedClusterPolicy != nil {
				expectedPolicies := &policyhierarchy_v1.AllPolicies{
					PolicyNodes:   tc.expectedPolicyNodes,
					ClusterPolicy: tc.expectedClusterPolicy,
				}

				if diff := deep.Equal(actualPolicies, expectedPolicies); diff != nil {
					t.Fatalf("Actual and expected policies didn't match: %v", diff)
				}
			}
		})
	}

}

// TODO(frankfarzan): Add a test for acme example.
