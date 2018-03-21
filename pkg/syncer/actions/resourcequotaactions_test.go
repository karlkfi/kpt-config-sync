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

	"github.com/google/nomos/pkg/client/meta/fake"
	"github.com/google/nomos/pkg/resourcequota"

	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listers_core_v1 "k8s.io/client-go/listers/core/v1"
)

type ResourceQuotaTestActionCtor func(
	string,
	map[string]string,
	core_v1.ResourceQuotaSpec,
	kubernetes.Interface,
	listers_core_v1.ResourceQuotaLister) ResourceQuotaAction
type ResourceQuotaTestCase struct {
	Name            string
	InitialNotFound bool
	InitialLabels   map[string]string
	InitialState    core_v1.ResourceQuotaSpec
	SpecifiedLabels map[string]string
	SpecifiedState  core_v1.ResourceQuotaSpec
	ExpectNotFound  bool
	ActionCtor      ResourceQuotaTestActionCtor

	// For testing that status is preserved on update.
	Status core_v1.ResourceQuotaStatus
}

// Namespace creates the unique namespace for the test based off of test index in the testcase slice
func (r *ResourceQuotaTestCase) Namespace(idx int) string {
	return fmt.Sprintf("test-namespace-%d", idx)
}

// NewResourceQuota creates a resoruce quota for the testcase in a namespace based off the test index
// so that each testcase is isolated to a unique namespace in the fake client.
func (r *ResourceQuotaTestCase) NewResourceQuota(idx int) *core_v1.ResourceQuota {
	r.Status = core_v1.ResourceQuotaStatus{
		Hard: r.InitialState.Hard,
		Used: core_v1.ResourceList{
			"pods": *resource.NewQuantity(int64(idx+5), resource.DecimalSI),
		},
	}

	return &core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      resourcequota.ResourceQuotaObjectName,
			Namespace: r.Namespace(idx),
			Labels:    r.InitialLabels,
		},
		Spec:   r.InitialState,
		Status: r.Status,
	}
}

func upsertQuotaTestAction(
	namespace string,
	labels map[string]string,
	resourceQuotaSpec core_v1.ResourceQuotaSpec,
	kubernetesInterface kubernetes.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister) ResourceQuotaAction {
	return NewResourceQuotaUpsertAction(
		namespace, labels, resourceQuotaSpec, kubernetesInterface, resourceQuotaLister)
}

func deleteQuotaTestAction(
	namespace string,
	labels map[string]string,
	resourceQuotaSpec core_v1.ResourceQuotaSpec,
	kubernetesInterface kubernetes.Interface,
	resourceQuotaLister listers_core_v1.ResourceQuotaLister) ResourceQuotaAction {
	return NewResourceQuotaDeleteAction(namespace, kubernetesInterface, resourceQuotaLister)
}

func newQuotaspec(amount int64) core_v1.ResourceQuotaSpec {
	return core_v1.ResourceQuotaSpec{
		Hard: core_v1.ResourceList{"pods": *resource.NewQuantity(amount, resource.DecimalSI)},
	}
}

var zeroPods = newQuotaspec(0)
var onePod = newQuotaspec(1)
var twoPods = newQuotaspec(2)

var resourceQuotaTestCases = []*ResourceQuotaTestCase{
	&ResourceQuotaTestCase{
		Name:            "Create during upsert",
		InitialNotFound: true,
		SpecifiedLabels: resourcequota.StolosQuotaLabels,
		SpecifiedState:  onePod,
		ActionCtor:      upsertQuotaTestAction,
	},
	&ResourceQuotaTestCase{
		Name:            "Update spec during upsert",
		InitialLabels:   resourcequota.StolosQuotaLabels,
		InitialState:    onePod,
		SpecifiedLabels: resourcequota.StolosQuotaLabels,
		SpecifiedState:  twoPods,
		ActionCtor:      upsertQuotaTestAction,
	},
	&ResourceQuotaTestCase{
		Name:            "Update labels during upsert",
		InitialLabels:   resourcequota.PolicySpaceQuotaLabels,
		InitialState:    zeroPods,
		SpecifiedLabels: resourcequota.StolosQuotaLabels,
		SpecifiedState:  zeroPods,
		ActionCtor:      upsertQuotaTestAction,
	},
	&ResourceQuotaTestCase{
		Name:            "No update during upsert",
		InitialLabels:   resourcequota.PolicySpaceQuotaLabels,
		InitialState:    zeroPods,
		SpecifiedLabels: resourcequota.PolicySpaceQuotaLabels,
		SpecifiedState:  zeroPods,
		ActionCtor:      upsertQuotaTestAction,
	},
	&ResourceQuotaTestCase{
		Name:            "Delete non existing item",
		InitialNotFound: true,
		ExpectNotFound:  true,
		ActionCtor:      deleteQuotaTestAction,
	},
	&ResourceQuotaTestCase{
		Name:           "Delete existing item",
		InitialLabels:  resourcequota.StolosQuotaLabels,
		InitialState:   twoPods,
		ExpectNotFound: true,
		ActionCtor:     deleteQuotaTestAction,
	},
}

func TestResourceQuotaActions(t *testing.T) {
	client := fake.NewClient()

	// Setup, each testcase gets it's own namespace
	for idx, testcase := range resourceQuotaTestCases {
		resourceQuota := client.Kubernetes().CoreV1().ResourceQuotas(testcase.Namespace(idx))
		// populate if needed
		if !testcase.InitialNotFound {
			_, err := resourceQuota.Create(testcase.NewResourceQuota(idx))
			if err != nil {
				t.Errorf("Failed to create initial state: %#v", err)
			}
		}
	}

	// Start informer factories, etc
	kubernetesInformerFactory := informers.NewSharedInformerFactory(
		client.Kubernetes(), time.Minute)
	quotaLister := kubernetesInformerFactory.Core().V1().ResourceQuotas().Lister()
	kubernetesInformerFactory.Start(nil)
	kubernetesInformerFactory.WaitForCacheSync(nil)

	for idx, testcase := range resourceQuotaTestCases {
		namespace := testcase.Namespace(idx)
		resourceQuota := client.Kubernetes().CoreV1().ResourceQuotas(namespace)

		// gen / execute action
		action := testcase.ActionCtor(
			namespace, testcase.SpecifiedLabels, testcase.SpecifiedState, client.Kubernetes(), quotaLister)
		err := action.Execute()
		if err != nil {
			t.Errorf("Failed to execute action %s", action)
		}

		// check postcondition
		actualQuota, err := resourceQuota.Get(resourcequota.ResourceQuotaObjectName, meta_v1.GetOptions{})
		if testcase.ExpectNotFound {
			if err != nil {
				if !api_errors.IsNotFound(err) {
					t.Errorf("Unexpected error during testcase")
				}
			} else {
				t.Errorf("Should have gotten not found error.")
			}
		} else {
			if err != nil {
				if api_errors.IsNotFound(err) {
					t.Errorf("Expected quota object to exist at end of testcase")
				}
				t.Errorf("Unexpected error during testcase")
			} else {
				// compare labels
				if !reflect.DeepEqual(testcase.SpecifiedState, actualQuota.Spec) {
					t.Errorf("Specified state does not match actual state")
				}
				// compare spec
				if !reflect.DeepEqual(testcase.SpecifiedLabels, actualQuota.ObjectMeta.Labels) {
					t.Errorf("Specified labels do not match actual labels")
				}
				if !testcase.InitialNotFound {
					// compare status
					if !reflect.DeepEqual(testcase.Status, actualQuota.Status) {
						t.Errorf("%d) Status was not preserved during update! %#v != %#v", idx, testcase.Status, actualQuota.Status)
					}
				}
			}
		}
	}
}
