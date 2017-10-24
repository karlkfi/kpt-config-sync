/*
Copyright 2017 The Kubernetes Authors.

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

package syncer

import (
	"testing"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta/fake"
	"github.com/google/stolos/pkg/client/policynodewatcher"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComputeResourceQuotaActionsTestCase struct {
	policyNodeResourceQuotas map[string]core_v1.ResourceQuotaSpec // Input policy node resource quota
	existingResourceQuotas   map[string]core_v1.ResourceQuotaSpec // Existing resource quotas in the cluster

	expectedActions map[string]string // A map of namespaces to the expected resource quota operation
}

func TestSyncerComputeResourceQuotaActions(t *testing.T) {
	syncer := NewQuotaSyncer(fake.NewClient())

	for i, testcase := range []ComputeResourceQuotaActionsTestCase{
		{ // Create
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{},
			expectedActions:        map[string]string{"foo": "create"},
		},
		{ // Update
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}},
			expectedActions: map[string]string{"foo": "update"},
		},
		{ // Delete
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}},
			expectedActions: map[string]string{"foo": "delete"},
		},
		{ // No diff
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			expectedActions: map[string]string{},
		},
	} {
		policyNodes := []*policyhierarchy_v1.PolicyNode{}
		for ns, rq := range testcase.policyNodeResourceQuotas {
			policyNodes = append(
				policyNodes,
				&policyhierarchy_v1.PolicyNode{
					ObjectMeta: meta_v1.ObjectMeta{Name: ns},
					Spec: policyhierarchy_v1.PolicyNodeSpec{
						WorkingNamespace: true,
						Policies:         policyhierarchy_v1.PolicyLists{ResourceQuotas: []core_v1.ResourceQuotaSpec{rq}}}})
		}

		existingResourceQuotaList := &core_v1.ResourceQuotaList{}
		for ns, rq := range testcase.existingResourceQuotas {
			existingResourceQuotaList.Items = append(
				existingResourceQuotaList.Items,
				core_v1.ResourceQuota{
					ObjectMeta: meta_v1.ObjectMeta{Name: ResourceQuotaObjectName, Namespace: ns},
					Spec:       rq},
			)
		}

		actions := syncer.computeActions(existingResourceQuotaList, policyNodes)
		if len(actions) != len(testcase.expectedActions) {
			t.Errorf("[T%d]: Expected %d actions but got %d", i, len(testcase.expectedActions), len(actions))
		}
		for _, action := range actions {
			if testcase.expectedActions[action.Name()] != action.Operation() {
				t.Errorf("[T%d]: Unexpected resource quota action %s %s", i, action.Operation(), action.Name())
			}
		}
	}
}

type GetResourceQuotaEventActionTestCase struct {
	event             policynodewatcher.EventType
	expectedOperation string
	noOperation       bool
	policyNode        policyhierarchy_v1.PolicyNode
}

func TestSyncerGetEventFesourceQuotaAction(t *testing.T) {
	syncer := NewQuotaSyncer(fake.NewClient())

	namespaceName := "ns-name"
	syncer.client.CoreV1().ResourceQuotas("ns-name").Create(&core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: ResourceQuotaObjectName,
		},
		Spec: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}},
	})

	policyNodeWithSameRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			WorkingNamespace: true,
			Policies: policyhierarchy_v1.PolicyLists{
				ResourceQuotas: []core_v1.ResourceQuotaSpec{
					{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}}},
		},
	}

	policyNodeWithDifferentRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			WorkingNamespace: true,
			Policies: policyhierarchy_v1.PolicyLists{
				ResourceQuotas: []core_v1.ResourceQuotaSpec{
					{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}}},
		},
	}
	policyNodeWithoutRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			WorkingNamespace: true,
		},
	}

	action := syncer.getCreateAction(&policyNodeWithSameRq)
	if action == nil {
		t.Error("Faild to make create action")
	}
	if action.Name() != namespaceName {
		t.Error("Wrong namespace name")
	}
	if action.Operation() != "create" {
		t.Error("Wrong operation")
	}

	action = syncer.getCreateAction(&policyNodeWithoutRq)
	if action != nil {
		t.Error("Should not have created action")
	}

	for idx, testcase := range []GetResourceQuotaEventActionTestCase{
		{event: policynodewatcher.Modified, expectedOperation: "delete", policyNode: policyNodeWithoutRq},
		{event: policynodewatcher.Modified, expectedOperation: "update", policyNode: policyNodeWithDifferentRq},
		{event: policynodewatcher.Modified, noOperation: true, policyNode: policyNodeWithSameRq},
	} {
		action, _ := syncer.getUpdateAction(&testcase.policyNode)
		if testcase.noOperation {
			if action != nil {
				t.Errorf("Got unexpected non-nil action %#v for testcase %d, data %#v", action, idx, testcase)
			}
			continue
		}
		if action == nil {
			t.Errorf("Got unexpected nil action for testcase %d, data %#v", idx, testcase)
			continue
		}
		if action.Name() != namespaceName {
			t.Errorf("Added event should have name %s, got %s", namespaceName, action.Name())
		}
		if action.Operation() != testcase.expectedOperation {
			t.Errorf("Got unexpected operation %s for testcase %d, data %#v", action.Operation(), idx, testcase)
		}
	}
}
