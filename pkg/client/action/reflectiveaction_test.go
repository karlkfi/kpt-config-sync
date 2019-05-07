/*
Copyright 2018 The CSP Config Management Authors.
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

package action

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/golang/mock/gomock"
	"github.com/google/nomos/clientgen/informer"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/meta/fake"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
)

type TypeGetter func(client *fake.Client, namespace, name string) (runtime.Object, error)
type ResourceFactory func(namespace, name string) runtime.Object

type ReflectiveActionTestCase struct {
	TestName string // Test name for human readability

	Namespaced      bool            // True if this resource exists at namespace level
	Operation       string          // Opertion to perform, "upsert" or "delete"
	Resource        ResourceFactory // resource factory function, needs to set name/namespace appropriately
	CreateExpectErr bool            // True if the create operation should expect an error

	// PrePopulate is a function that takes the original resource and returns nil or a resource that
	// should be pre-populated on the host.
	PrePopulate func(runtime.Object) runtime.Object

	// Calls get for the resource under test and returns it as an object.
	Getter TypeGetter

	// expectSkipDelete is true when delete actions are expected to not complete.
	expectSkipDelete bool

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
	switch t.Operation {
	case "create":
		if t.CreateExpectErr {
			if err == nil {
				test.Errorf("Expected create error in testcase %s: %s", t.TestName, err)
				return
			}
		} else {
			if err != nil {
				test.Errorf("Encountered error in testcase %s: %s", t.TestName, err)
				return
			}
		}
	case "upsert", "delete":
		if err != nil {
			test.Errorf("Encountered error in testcase %s: %s", t.TestName, err)
			return
		}
	}

	if msg := t.Validate(); msg != "" {
		test.Errorf("Validation failed on testcase %s: %s", t.TestName, msg)
	}
}

func (t *ReflectiveActionTestCase) Validate() string {
	glog.Infof("Checking testcase %s", t.TestName)
	switch t.Operation {
	case "create":
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
		if !t.CreateExpectErr {
			if !t.spec.EqualSpec(obj, resource) {
				return fmt.Sprintf("Objects %v and %v differ", spew.Sdump(obj), spew.Sdump(resource))
			}
			if !ObjectMetaSubset(obj, resource) {
				return fmt.Sprintf("Objects %v and %v differ", spew.Sdump(obj), spew.Sdump(resource))
			}
		}
		return ""

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
			if t.expectSkipDelete {
				// We don't expect the delete to go through for this test case.
				return ""
			}
			return "Resource should have been deleted"
		}
		if err != nil && apierrors.IsNotFound(err) {
			return ""
		}
		return fmt.Sprintf("Error other than not found: %s", err)
	default:
		panic(fmt.Sprintf("Invalid operation %s", t.Operation))
	}
}

func (t *ReflectiveActionTestCase) CreateOperation() Interface {
	switch t.Operation {
	case "create":
		return NewReflectiveCreateAction(
			t.Namespace(), t.Name(), t.GetResource(), t.spec, nil)
	case "upsert":
		return NewReflectiveUpsertAction(
			t.Namespace(), t.Name(), t.GetResource(), t.spec, nil)
	case "delete":
		return NewReflectiveDeleteAction(t.Namespace(), t.Name(), t.spec, nil)
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
	var storage []runtime.Object
	for _, testcase := range t.testcases {
		resource := testcase.GetPrePopulated()
		if resource == nil {
			continue
		}
		storage = append(storage, resource.DeepCopyObject())
	}
	mockCtrl := gomock.NewController(t.t)
	defer mockCtrl.Finish()

	mockClient := mocks.NewMockClient(mockCtrl)
	t.client = fake.NewClientWithStorage(storage, mockClient)

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
	lRole := lhs.(*rbacv1.Role)
	rRole := rhs.(*rbacv1.Role)
	return reflect.DeepEqual(lRole.Rules, rRole.Rules)
}

func RoleGetter(client *fake.Client, namespace, name string) (runtime.Object, error) {
	return client.Kubernetes().RbacV1().Roles(namespace).Get(name, metav1.GetOptions{})
}

var namespacedBaseTestObject = &rbacv1.Role{
	TypeMeta: metav1.TypeMeta{
		Kind:       "Role",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Labels:      map[string]string{"fuzzy": "true"},
		Annotations: map[string]string{"api.foo.future/deny": "*"},
	},
	Rules: []rbacv1.PolicyRule{
		{
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
	{
		TestName:    "create ok",
		Operation:   "create",
		Namespaced:  true,
		Resource:    namespacedRoleX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	{
		TestName:        "create fails",
		Operation:       "create",
		Namespaced:      true,
		CreateExpectErr: true,
		Resource:        namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.Role).DeepCopy()
			role.ObjectMeta.Labels["foo"] = "bar"
			return role
		},
	},
	{
		TestName:    "upsert for create",
		Operation:   "upsert",
		Namespaced:  true,
		Resource:    namespacedRoleX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	{
		TestName:   "upsert to different labels",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.Role).DeepCopy()
			role.ObjectMeta.Labels["foo"] = "bar"
			return role
		},
	},
	{
		TestName:   "upsert to different annotations",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.Role).DeepCopy()
			role.ObjectMeta.Annotations["foo"] = "bar"
			return role
		},
	},
	{
		TestName:   "upsert to different content",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.Role).DeepCopy()
			role.Rules = append(role.Rules, rbacv1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{""},
				Resources: []string{"*"},
			})
			return role
		},
	},
	{
		TestName:   "upsert to identical",
		Operation:  "upsert",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	{
		TestName:   "delete existing",
		Operation:  "delete",
		Namespaced: true,
		Resource:   namespacedRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	{
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
	{
		TestName:    "create ok",
		Operation:   "create",
		Resource:    clusterRoleX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	{
		TestName:        "create fails",
		Operation:       "create",
		CreateExpectErr: true,
		Resource:        clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.ClusterRole).DeepCopy()
			role.ObjectMeta.Labels["foo"] = "bar"
			return role
		},
	},
	{
		TestName:    "upsert for create",
		Operation:   "upsert",
		Resource:    clusterRoleX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	{
		TestName:  "upsert to different labels",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.ClusterRole).DeepCopy()
			role.ObjectMeta.Labels["foo"] = "bar"
			return role
		},
	},
	{
		TestName:  "upsert to different annotations",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.ClusterRole).DeepCopy()
			role.ObjectMeta.Annotations["foo"] = "bar"
			return role
		},
	},
	{
		TestName:  "upsert to different content",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			role := obj.(*rbacv1.ClusterRole).DeepCopy()
			role.Rules = append(role.Rules, rbacv1.PolicyRule{
				Verbs:     []string{"*"},
				APIGroups: []string{""},
				Resources: []string{"*"},
			})
			return role
		},
	},
	{
		TestName:  "upsert to identical",
		Operation: "upsert",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	{
		TestName:  "delete existing",
		Operation: "delete",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	{
		TestName:  "delete finalizing",
		Operation: "delete",
		Resource:  clusterRoleX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			o := obj.DeepCopyObject()
			m := o.(metav1.Object)
			ts := metav1.Now()
			m.SetDeletionTimestamp(&ts)
			return m.(runtime.Object)
		},
		expectSkipDelete: true,
	},
	{
		TestName:  "delete does not exist",
		Operation: "delete",
	},
}

func ClusterRolesEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lRole := lhs.(*rbacv1.ClusterRole)
	rRole := rhs.(*rbacv1.ClusterRole)
	return reflect.DeepEqual(lRole.Rules, rRole.Rules)
}

func ClusterRoleGetter(client *fake.Client, _, name string) (runtime.Object, error) {
	return client.Kubernetes().RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
}

var clusterBaseTestObject = &rbacv1.ClusterRole{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ClusterRole",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Labels:      map[string]string{"fuzzy": "true"},
		Annotations: map[string]string{"api.foo.future/deny": "*"},
	},
	Rules: []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get"},
			APIGroups: []string{"some.group.k8s.io"},
			Resources: []string{"*"},
		},
	},
}

func clusterRoleX(_, name string) runtime.Object {
	testRole := clusterBaseTestObject.DeepCopy()
	testRole.Name = name
	return testRole
}

func TestReflectiveActionCluster(t *testing.T) {
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

var clusterNamespaceConfigTestCases = []ReflectiveActionTestCase{
	{
		TestName:    "upsert for create",
		Operation:   "upsert",
		Resource:    clusterNamespaceConfigX,
		PrePopulate: func(runtime.Object) runtime.Object { return nil },
	},
	{
		TestName:  "upsert to different labels",
		Operation: "upsert",
		Resource:  clusterNamespaceConfigX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			namespaceConfig := obj.(*v1.NamespaceConfig).DeepCopy()
			namespaceConfig.ObjectMeta.Labels["foo"] = "bar"
			return namespaceConfig
		},
	},
	{
		TestName:  "upsert to different annotations",
		Operation: "upsert",
		Resource:  clusterNamespaceConfigX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			namespaceConfig := obj.(*v1.NamespaceConfig).DeepCopy()
			namespaceConfig.ObjectMeta.Annotations["foo"] = "bar"
			return namespaceConfig
		},
	},
	{
		TestName:  "upsert to different content",
		Operation: "upsert",
		Resource:  clusterNamespaceConfigX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			namespaceConfig := obj.(*v1.NamespaceConfig).DeepCopy()
			return namespaceConfig
		},
	},
	{
		TestName:  "upsert to identical",
		Operation: "upsert",
		Resource:  clusterNamespaceConfigX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	{
		TestName:  "delete existing",
		Operation: "delete",
		Resource:  clusterNamespaceConfigX,
		PrePopulate: func(obj runtime.Object) runtime.Object {
			return obj.DeepCopyObject()
		},
	},
	{
		TestName:  "delete does not exist",
		Operation: "delete",
	},
}

func NamespaceConfigsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	lPn := lhs.(*v1.NamespaceConfig)
	rPn := rhs.(*v1.NamespaceConfig)
	return reflect.DeepEqual(lPn.Spec, rPn.Spec)
}

func ClusterConfigNodeGetter(client *fake.Client, _, name string) (runtime.Object, error) {
	return client.ConfigManagement().ConfigmanagementV1().NamespaceConfigs().Get(name, metav1.GetOptions{})
}

var clusterNamespaceConfigBaseTestObject = &v1.NamespaceConfig{
	TypeMeta: metav1.TypeMeta{
		Kind:       "NamespaceConfig",
		APIVersion: v1.SchemeGroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Labels:      map[string]string{"fuzzy": "true"},
		Annotations: map[string]string{"api.foo.future/deny": "*"},
	},
	Spec: v1.NamespaceConfigSpec{},
}

func clusterNamespaceConfigX(_, name string) runtime.Object {
	testObject := clusterNamespaceConfigBaseTestObject.DeepCopy()
	testObject.Name = name
	return testObject
}

func TestReflectiveActionNamespaceConfig(t *testing.T) {
	test := NewReflectiveActionTest(t, clusterNamespaceConfigTestCases)

	test.PrePopulate()

	factory := informer.NewSharedInformerFactory(test.client.ConfigManagement(), time.Second*10)

	spec := &ReflectiveActionSpec{
		KindPlural: "NamespaceConfigs",
		EqualSpec:  NamespaceConfigsEqual,
		Client:     test.client.ConfigManagement().ConfigmanagementV1(),
		Lister:     factory.Configmanagement().V1().NamespaceConfigs().Lister(),
	}

	factory.Start(nil)
	factory.WaitForCacheSync(nil)

	test.RunTests(spec, ClusterConfigNodeGetter)
}
