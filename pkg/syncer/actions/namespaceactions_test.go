/*
Copyright 2017 The Stolos Authors.
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
package actions

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/google/stolos/pkg/client/meta/fake"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

const parentLabelValue = "parent-team"
const testUID = types.UID("0844e3b3-1059-11e8-9233-42010a800005")

func upsertNamespaceTestAction(ns string, client kubernetes.Interface, nsLister listers_core_v1.NamespaceLister) Interface {
	return NewNamespaceUpsertAction(
		ns,
		testUID,
		map[string]string{policyhierarchy_v1.ParentLabelKey: parentLabelValue},
		client,
		nsLister)
}

func deleteNamespaceTestAction(
	ns string, client kubernetes.Interface, nsLister listers_core_v1.NamespaceLister) Interface {
	return NewNamespaceDeleteAction(ns, client, nsLister)
}

type NamespaceActionCtor func(ns string, client kubernetes.Interface, nsLister listers_core_v1.NamespaceLister) Interface
type NamespaceTestCase struct {
	Name         string
	Exists       bool
	ActionCtor   NamespaceActionCtor
	ExpectExists bool
}

func (s *NamespaceTestCase) Namespace(idx int) string {
	return fmt.Sprintf("namespace-%d", idx)
}

var namespaceTestCases = []NamespaceTestCase{
	NamespaceTestCase{
		Name:         "Create non-existing namespace",
		Exists:       false,
		ActionCtor:   upsertNamespaceTestAction,
		ExpectExists: true,
	},
	NamespaceTestCase{
		Name:         "Update existing namespace",
		Exists:       true,
		ActionCtor:   upsertNamespaceTestAction,
		ExpectExists: true,
	},
	NamespaceTestCase{
		Name:         "Delete non-existing namespace",
		Exists:       false,
		ActionCtor:   deleteNamespaceTestAction,
		ExpectExists: false,
	},
	NamespaceTestCase{
		Name:         "Delete existing namespace",
		Exists:       true,
		ActionCtor:   deleteNamespaceTestAction,
		ExpectExists: false,
	},
}

func TestNamespaceActions(t *testing.T) {
	client := fake.NewClient()
	nsClient := client.Kubernetes().CoreV1().Namespaces()

	// Setup, each testcase gets it's own namespace
	for idx, testcase := range namespaceTestCases {
		if testcase.Exists {
			_, err := nsClient.Create(&core_v1.Namespace{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: testcase.Namespace(idx),
				}})
			if err != nil {
				t.Errorf("Failed to create initial state: %#v", err)
			}
		}
	}

	kubernetesInformerFactory := informers.NewSharedInformerFactory(
		client.Kubernetes(), time.Minute)
	namespaceLister := kubernetesInformerFactory.Core().V1().Namespaces().Lister()
	kubernetesInformerFactory.Start(nil)
	kubernetesInformerFactory.WaitForCacheSync(nil)

	for idx, testcase := range namespaceTestCases {
		namespace := testcase.Namespace(idx)

		action := testcase.ActionCtor(namespace, client.Kubernetes(), namespaceLister)
		err := action.Execute()
		if err != nil {
			t.Errorf("Failed to execute action %s", action)
		}

		ns, err := nsClient.Get(namespace, meta_v1.GetOptions{})
		if testcase.ExpectExists {
			if err != nil {
				if api_errors.IsNotFound(err) {
					t.Errorf("Testcase should have created namespace")
				}
				t.Errorf("Unexpected error during testcase")
			}
			if ns.Labels[policyhierarchy_v1.ParentLabelKey] != parentLabelValue {
				t.Errorf("Failed to update the parent label\ntestcase: %s\n %s", spew.Sdump(testcase), spew.Sdump(ns))
			}
			blockOwnerDeletion := true
			expectOwnerReferences := []meta_v1.OwnerReference{
				meta_v1.OwnerReference{
					APIVersion:         policyhierarchy_v1.SchemeGroupVersion.String(),
					Kind:               "PolicyNode",
					Name:               namespace,
					UID:                testUID,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			}
			if !reflect.DeepEqual(expectOwnerReferences, ns.OwnerReferences) {
				t.Errorf("Failed to set owner reference on namespace %s != %s",
					spew.Sdump(expectOwnerReferences), spew.Sdump(ns.OwnerReferences))
			}
		} else {
			if err != nil {
				if !api_errors.IsNotFound(err) {
					t.Errorf("Unexpected error during testcase")
				}
			} else {
				t.Errorf("Namespace should not exist at end of testcase.")
			}
		}
	}
}
