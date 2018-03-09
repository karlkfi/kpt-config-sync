package action

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/glog"

	"github.com/davecgh/go-spew/spew"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	"github.com/google/stolos/pkg/client/meta/fake"
	rbac_v1 "k8s.io/api/rbac/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
)

type TypeGetter func(client *fake.Client, namespace, name string) (runtime.Object, error)
type ResourceFactory func(namespace, name string) runtime.Object

type ReflectiveActionTestCase struct {
	TestName string // Test name for human readability

	Namespaced bool            // True if this resource exists at namespace level
	Operation  string          // Opertion to perform, "upsert" or "delete"
	Resource   ResourceFactory // resource factory function, needs to set name/namespace appropraitely

	// PrePopulate is a function that takes the original resource and returns nil or a resource that
	// should be pre-populated on the host.
	PrePopulate func(runtime.Object) runtime.Object

	// Calls get for the resource under test and returns it as an object.
	Getter TypeGetter

	// Provided by test harness
	idx    int
	spec   *ReflectiveActionSpec
	client *fake.Client
}

// GetResource gets the resource from the factory or returns nil if one is not defined
func (t *ReflectiveActionTestCase) GetResource() runtime.Object {
	if t.Resource == nil {
		return nil
	}
	return t.Resource(t.Namespace(), t.Name())
}

func (t *ReflectiveActionTestCase) GetPrePopulated() runtime.Object {
	res := t.GetResource()
	if res == nil || t.PrePopulate == nil {
		return nil
	}
	return t.PrePopulate(res)
}

func (t *ReflectiveActionTestCase) Namespace() string {
	if t.Namespaced {
		return fmt.Sprintf("test-resource-%d", t.idx)
	}
	return ""
}

func (t *ReflectiveActionTestCase) Name() string {
	return fmt.Sprintf("test-ns-%d", t.idx)
}

func (t *ReflectiveActionTestCase) Run(test *testing.T) {
	operation := t.CreateOperation()
	err := operation.Execute()
	if err != nil {
		test.Errorf("Encountered error in testcase %s: %s", t.TestName, err)
		return
	}

	if msg := t.Validate(); msg != "" {
		test.Errorf("Validation failed on testcase %s: %s", t.TestName, msg)
	}
}

func (t *ReflectiveActionTestCase) Validate() string {
	glog.Infof("Checking testcase %s", t.TestName)
	switch t.Operation {
	case "upsert":
		// check resource exists and
		resource := t.GetResource()
		obj, err := t.Getter(t.client, t.Namespace(), t.Name())
		if err != nil {
			return fmt.Sprintf("Failed to get object: %s", err)
		}
		if resource == nil {
			panic("resource is nil")
		}
		if obj == nil {
			panic("object is nil")
		}
		if !t.spec.EqualSpec(obj, resource) {
			return fmt.Sprintf("Objects %v and %v differ", spew.Sdump(obj), spew.Sdump(resource))
		}
		if !ObjectMetaSubset(obj, resource) {
			return fmt.Sprintf("Objects %v and %v differ", spew.Sdump(obj), spew.Sdump(resource))
		}
		return ""

	case "delete":
		// check resource does not exist
		_, err := t.Getter(t.client, t.Namespace(), t.Name())
		if err == nil {
			return "Resource should have been deleted"
		}
		if err != nil && api_errors.IsNotFound(err) {
			return ""
		}
		return fmt.Sprintf("Error other than not found: %s", err)
	default:
		panic(fmt.Sprintf("Invalid operation %s", t.Operation))
	}
}

func (t *ReflectiveActionTestCase) CreateOperation() Interface {
	switch t.Operation {
	case "upsert":
		return NewReflectiveUpsertAction(
			t.Namespace(), t.Name(), t.GetResource(), t.spec)
	case "delete":
		return NewReflectiveDeleteAction(t.Namespace(), t.Name(), t.spec)
	default:
		panic(fmt.Sprintf("Invalid operation %s", t.Operation))
	}
}

type ReflectiveActionTest struct {
	t         *testing.T
	client    *fake.Client
	spec      *ReflectiveActionSpec
	testcases []ReflectiveActionTestCase
}

func NewReflectiveActionTest(
	t *testing.T, testcases []ReflectiveActionTestCase) *ReflectiveActionTest {
	for idx := range testcases {
		testcases[idx].idx = idx
	}
	return &ReflectiveActionTest{
		t:         t,
		testcases: testcases,
	}
}

func (t *ReflectiveActionTest) PrePopulate() {
	var kubernetesStorage, policyHierarchyStorage []runtime.Object
	for _, testcase := range t.testcases {
		resource := testcase.GetPrePopulated()
		if resource == nil {
			continue
		}

		if resource.GetObjectKind().GroupVersionKind().Group ==
			policyhierarchy_v1.SchemeGroupVersion.Group {
			policyHierarchyStorage = append(policyHierarchyStorage, resource.DeepCopyObject())
		} else {
			kubernetesStorage = append(kubernetesStorage, resource.DeepCopyObject())
		}
	}
	t.client = fake.NewClientWithStorage(kubernetesStorage, policyHierarchyStorage)

	for idx := range t.testcases {
		t.testcases[idx].client = t.client
	}
}

func (t *ReflectiveActionTest) RunTests(spec *ReflectiveActionSpec, getter TypeGetter) {
	t.spec = spec
	for idx := range t.testcases {
		t.testcases[idx].spec = spec
		t.testcases[idx].Getter = getter
	}

	for _, testcase := range t.testcases {
		t.t.Run(testcase.TestName, testcase.Run)
	}
}

func RolesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRole := lhs.(*rbac_v1.Role)
	rRole := rhs.(*rbac_v1.Role)
	return reflect.DeepEqual(lRole.Rules, rRole.Rules)
}

func RoleGetter(client *fake.Client, namespace, name string) (runtime.Object, error) {
	return client.Kubernetes().RbacV1().Roles(namespace).Get(name, meta_v1.GetOptions{})
}

var namespacedBaseTestObject = &rbac_v1.Role{
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: meta_v1.ObjectMeta{
		Labels:      map[string]string{"fuzzy": "true"},
		Annotations: map[string]string{"api.foo.future/deny": "*"},
	},
	Rules: []rbac_v1.PolicyRule{
		rbac_v1.PolicyRule{
			Verbs:     []string{"get"},
			APIGroups: []string{"some.group.k8s.io"},
			Resources: []string{"*"},
		},
	},
}

func namespacedRoleX(namespace, name string) runtime.Object {
	testRole := namespacedBaseTestObject.DeepCopy()
	testRole.Namespace = namespace
	testRole.Name = name
	return testRole
}

var namespacedTestCases = []ReflectiveActionTestCase{
	ReflectiveActionTestCase{
		TestName:    "upsert for create",
		Operation:   "upsert",
		Namespaced:  true,
		Resource:    namespacedRoleX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	ReflectiveActionTestCase{
		TestName:   "upsert to different labels",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbac_v1.Role).DeepCopy()
			role.ObjectMeta.Labels["foo"] = "bar"
			return role
		},
	},
	ReflectiveActionTestCase{
		TestName:   "upsert to different annotations",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbac_v1.Role).DeepCopy()
			role.ObjectMeta.Annotations["foo"] = "bar"
			return role
		},
	},
	ReflectiveActionTestCase{
		TestName:   "upsert to different content",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbac_v1.Role).DeepCopy()
			role.Rules = append(role.Rules, rbac_v1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{""},
				Resources: []string{"*"},
			})
			return role
		},
	},
	ReflectiveActionTestCase{
		TestName:   "upsert to identical",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	ReflectiveActionTestCase{
		TestName:   "delete existing",
		Operation:  "delete",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	ReflectiveActionTestCase{
		TestName:   "delete does not exist",
		Operation:  "delete",
		Namespaced: true,
	},
}

func TestReflectiveActionNamespaced(t *testing.T) {
	test := NewReflectiveActionTest(t, namespacedTestCases)

	test.PrePopulate()

	factory := informers.NewSharedInformerFactory(test.client.Kubernetes(), time.Second*10)

	spec := &ReflectiveActionSpec{
		KindPlural: "Roles",
		EqualSpec:  RolesEqual,
		Client:     test.client.Kubernetes().RbacV1(),
		Lister:     factory.Rbac().V1().Roles().Lister(),
	}

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	test.RunTests(spec, RoleGetter)
}

var clusterTestCases = []ReflectiveActionTestCase{
	ReflectiveActionTestCase{
		TestName:    "upsert for create",
		Operation:   "upsert",
		Resource:    clusterRoleX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to different labels",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbac_v1.ClusterRole).DeepCopy()
			role.ObjectMeta.Labels["foo"] = "bar"
			return role
		},
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to different annotations",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbac_v1.ClusterRole).DeepCopy()
			role.ObjectMeta.Annotations["foo"] = "bar"
			return role
		},
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to different content",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbac_v1.ClusterRole).DeepCopy()
			role.Rules = append(role.Rules, rbac_v1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{""},
				Resources: []string{"*"},
			})
			return role
		},
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to identical",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	ReflectiveActionTestCase{
		TestName:  "delete existing",
		Operation: "delete",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	ReflectiveActionTestCase{
		TestName:  "delete does not exist",
		Operation: "delete",
	},
}

func ClusterRolesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRole := lhs.(*rbac_v1.ClusterRole)
	rRole := rhs.(*rbac_v1.ClusterRole)
	return reflect.DeepEqual(lRole.Rules, rRole.Rules)
}

func ClusterRoleGetter(client *fake.Client, namespace, name string) (runtime.Object, error) {
	return client.Kubernetes().RbacV1().ClusterRoles().Get(name, meta_v1.GetOptions{})
}

var clusterBaseTestObject = &rbac_v1.ClusterRole{
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "",
		APIVersion: "",
	},
	ObjectMeta: meta_v1.ObjectMeta{
		Labels:      map[string]string{"fuzzy": "true"},
		Annotations: map[string]string{"api.foo.future/deny": "*"},
	},
	Rules: []rbac_v1.PolicyRule{
		rbac_v1.PolicyRule{
			Verbs:     []string{"get"},
			APIGroups: []string{"some.group.k8s.io"},
			Resources: []string{"*"},
		},
	},
}

func clusterRoleX(namespace, name string) runtime.Object {
	testRole := clusterBaseTestObject.DeepCopy()
	testRole.Name = name
	return testRole
}

func TestRefelctiveActionCluster(t *testing.T) {
	test := NewReflectiveActionTest(t, clusterTestCases)

	test.PrePopulate()

	factory := informers.NewSharedInformerFactory(test.client.Kubernetes(), time.Second*10)

	spec := &ReflectiveActionSpec{
		KindPlural: "ClusterRoles",
		EqualSpec:  ClusterRolesEqual,
		Client:     test.client.Kubernetes().RbacV1(),
		Lister:     factory.Rbac().V1().ClusterRoles().Lister(),
	}

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	test.RunTests(spec, ClusterRoleGetter)
}

var clusterPolicyNodeTestCases = []ReflectiveActionTestCase{
	ReflectiveActionTestCase{
		TestName:    "upsert for create",
		Operation:   "upsert",
		Resource:    clusterPolicyNodeX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to different labels",
		Operation: "upsert",
		Resource:  clusterPolicyNodeX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			policyNode := obj.(*policyhierarchy_v1.PolicyNode).DeepCopy()
			policyNode.ObjectMeta.Labels["foo"] = "bar"
			return policyNode
		},
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to different annotations",
		Operation: "upsert",
		Resource:  clusterPolicyNodeX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			policyNode := obj.(*policyhierarchy_v1.PolicyNode).DeepCopy()
			policyNode.ObjectMeta.Annotations["foo"] = "bar"
			return policyNode
		},
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to different content",
		Operation: "upsert",
		Resource:  clusterPolicyNodeX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			policyNode := obj.(*policyhierarchy_v1.PolicyNode).DeepCopy()
			policyNode.Spec.Parent = "some-other-node"
			return policyNode
		},
	},
	ReflectiveActionTestCase{
		TestName:  "upsert to identical",
		Operation: "upsert",
		Resource:  clusterPolicyNodeX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	ReflectiveActionTestCase{
		TestName:  "delete existing",
		Operation: "delete",
		Resource:  clusterPolicyNodeX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	ReflectiveActionTestCase{
		TestName:  "delete does not exist",
		Operation: "delete",
	},
}

func PolicyNodesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lPn := lhs.(*policyhierarchy_v1.PolicyNode)
	rPn := rhs.(*policyhierarchy_v1.PolicyNode)
	return reflect.DeepEqual(lPn.Spec, rPn.Spec)
}

func ClusterPolicyNodeGetter(client *fake.Client, namespace, name string) (runtime.Object, error) {
	return client.PolicyHierarchy().StolosV1().PolicyNodes().Get(name, meta_v1.GetOptions{})
}

var clusterPolicyNodeBaseTestObject = &policyhierarchy_v1.PolicyNode{
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "PolicyNode",
		APIVersion: policyhierarchy_v1.SchemeGroupVersion.String(),
	},
	ObjectMeta: meta_v1.ObjectMeta{
		Labels:      map[string]string{"fuzzy": "true"},
		Annotations: map[string]string{"api.foo.future/deny": "*"},
	},
	Spec: policyhierarchy_v1.PolicyNodeSpec{
		Policyspace: true,
		Parent:      "does-not-exist",
	},
}

func clusterPolicyNodeX(namespace, name string) runtime.Object {
	testObject := clusterPolicyNodeBaseTestObject.DeepCopy()
	testObject.Name = name
	return testObject
}

func TestRefelctiveActionPolicyNode(t *testing.T) {
	test := NewReflectiveActionTest(t, clusterPolicyNodeTestCases)

	test.PrePopulate()

	factory := externalversions.NewSharedInformerFactory(test.client.PolicyHierarchy(), time.Second*10)

	spec := &ReflectiveActionSpec{
		KindPlural: "PolicyNodes",
		EqualSpec:  PolicyNodesEqual,
		Client:     test.client.PolicyHierarchy().StolosV1(),
		Lister:     factory.Stolos().V1().PolicyNodes().Lister(),
	}

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	test.RunTests(spec, ClusterPolicyNodeGetter)
}

type ObjectMetaSubsetTestcase struct {
	Name         string
	Set          meta_v1.ObjectMeta
	Subset       meta_v1.ObjectMeta
	ExpectReturn bool
}

func newObjectMeta(labels map[string]string, annotations map[string]string) meta_v1.ObjectMeta {
	return meta_v1.ObjectMeta{
		Labels:      labels,
		Annotations: annotations,
	}
}

func (tc *ObjectMetaSubsetTestcase) Run(t *testing.T) {
	var Set, Subset runtime.Object
	Set = &rbac_v1.Role{ObjectMeta: tc.Set}
	Subset = &rbac_v1.Role{ObjectMeta: tc.Subset}
	if ObjectMetaSubset(Set, Subset) != tc.ExpectReturn {
		t.Errorf("Testcase Failure %v", tc)
	}
}

var objectMetaTestcases = []ObjectMetaSubsetTestcase{
	ObjectMetaSubsetTestcase{
		Name: "labels and annotations are both subsets",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: true,
	},
	ObjectMetaSubsetTestcase{
		Name: "labels not subset",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	ObjectMetaSubsetTestcase{
		Name: "annotations not subset",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k5": "v5"},
		),
		ExpectReturn: false,
	},
	ObjectMetaSubsetTestcase{
		Name: "neither are subset",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k6": "v6"},
		),
		ExpectReturn: false,
	},
}

func TestObjectMetaSubset(t *testing.T) {
	for _, testcase := range objectMetaTestcases {
		t.Run(testcase.Name, testcase.Run)
	}
}

type IsSubsetTestcase struct {
	Name         string
	Set          map[string]string
	Subset       map[string]string
	ExpectReturn bool
}

func (tc *IsSubsetTestcase) Run(t *testing.T) {
	if isSubset(tc.Set, tc.Subset) != tc.ExpectReturn {
		t.Errorf("Testcase Failure %v", tc)
	}
}

var isSubsetTestcases = []IsSubsetTestcase{
	IsSubsetTestcase{
		Name:         "both nil/emtpy",
		Set:          nil,
		Subset:       nil,
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "subset nil/empty",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       nil,
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "set nil/empty",
		Set:          nil,
		Subset:       map[string]string{"k1": "v1", "k2": "v2"},
		ExpectReturn: false,
	},
	IsSubsetTestcase{
		Name:         "subset is subset",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1"},
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "both equivalent",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1", "k2": "v2"},
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "subset has extra elements equivalent",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
		ExpectReturn: false,
	},
	IsSubsetTestcase{
		Name:         "subset is not subset",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k3": "v3"},
		ExpectReturn: false,
	},
	IsSubsetTestcase{
		Name:         "subset has different value",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1", "k2": "mismatch"},
		ExpectReturn: false,
	},
}

func TestIsSubset(t *testing.T) {
	for _, testcase := range isSubsetTestcases {
		t.Run(testcase.Name, testcase.Run)
	}
}
