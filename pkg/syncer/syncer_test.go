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
	"github.com/google/stolos/pkg/util/set/stringset"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ComputeNamespaceActionsTestCase struct {
	policyNodeNamespaces  []string // namespaces defined in the poicy node objects
	existingNamespaces    []string // namespaces in the active state
	terminatingNamespaces []string // namespaces in the "terminating" state

	needsCreate []string // namespaces that will be deleted
	needsDelete []string // namespaces that will be created
}

func createNamespace(name string, phase core_v1.NamespacePhase) core_v1.Namespace {
	return core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{Name: name},
		Status:     core_v1.NamespaceStatus{Phase: phase},
	}
}

func newTestSyncer() *Syncer {
	return &Syncer{client: fake.NewClient()}
}

func TestSyncerComputeNamespaceActions(t *testing.T) {
	syncer := newTestSyncer()

	for _, testcase := range []ComputeNamespaceActionsTestCase{
		{// Create terminating ns
			policyNodeNamespaces: []string{"foo"},
			existingNamespaces: []string{"bar"},
			terminatingNamespaces: []string{"foo"},
			needsCreate: []string{"foo"},
			needsDelete: []string{"bar"},
		},
		{// need create, need delete
			policyNodeNamespaces: []string{"foo", "foo2"},
			existingNamespaces: []string{"bar", "bar2"},
			terminatingNamespaces: []string{"baz"},
			needsCreate: []string{"foo", "foo2"},
			needsDelete: []string{"bar", "bar2"},
		},
		{// need create
			policyNodeNamespaces: []string{"foo", "bar"},
			existingNamespaces: []string{"bar"},
			terminatingNamespaces: []string{},
			needsCreate: []string{"foo"},
			needsDelete: []string{},
		},
		{// need delete
			policyNodeNamespaces: []string{"foo"},
			existingNamespaces: []string{"bar", "foo"},
			terminatingNamespaces: []string{"baz"},
			needsCreate: []string{},
			needsDelete: []string{"bar"},
		},
		{// No diff
			policyNodeNamespaces: []string{"foo"},
			existingNamespaces: []string{"foo"},
			terminatingNamespaces: []string{"baz"},
			needsCreate: []string{},
			needsDelete: []string{},
		},
	} {
		policyNodeList := &policyhierarchy_v1.PolicyNodeList{}
		for _, value := range testcase.policyNodeNamespaces {
			policyNodeList.Items = append(
				policyNodeList.Items,
				policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: value}})
		}

		namespaceList := &core_v1.NamespaceList{}
		for _, value := range testcase.existingNamespaces {
			namespaceList.Items = append(namespaceList.Items, createNamespace(value, core_v1.NamespaceActive))
		}
		for _, value := range testcase.terminatingNamespaces {
			namespaceList.Items = append(namespaceList.Items, createNamespace(value, core_v1.NamespaceTerminating))
		}

		actions := syncer.computeNamespaceActionsWithNamespaceList(namespaceList, policyNodeList)

		nsCreate := stringset.New()
		nsDelete := stringset.New()
		for _, action := range actions {
			switch action.Operation() {
			case "create":
				nsCreate.Add(action.Name())
			case "delete":
				nsDelete.Add(action.Name())
			default:
				t.Errorf("Got invalid action operation %s", action.Operation())
			}
		}
		expectedCreate := stringset.NewFromSlice(testcase.needsCreate)
		expectedDelete := stringset.NewFromSlice(testcase.needsDelete)

		if !nsCreate.Equals(expectedCreate) {
			t.Errorf("Expected creations to be %v but got %v", expectedCreate, nsCreate)
		}
		if !nsDelete.Equals(expectedDelete) {
			t.Errorf("Expected deletions to be %v but got %v", expectedDelete, nsDelete)
		}
	}
}

type ComputeResourceQuotaActionsTestCase struct {
	policyNodeResourceQuotas map[string]core_v1.ResourceQuotaSpec // Input policy node resource quota
	existingResourceQuotas   map[string]core_v1.ResourceQuotaSpec // Existing resource quotas in the cluster

	expectedActions map[string]string // A map of namespaces to the expected resource quota operation
}

func TestSyncerComputeResourceQuotaActions(t *testing.T) {
	syncer := newTestSyncer()

	for i, testcase := range []ComputeResourceQuotaActionsTestCase{
		{// Create
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{},
			expectedActions: map[string]string{"foo": "create"},
		},
		{// Update
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}},
			expectedActions: map[string]string{"foo": "update"},
		},
		{// Delete
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}},
			expectedActions: map[string]string{"foo": "delete"},
		},
		{// No diff
			policyNodeResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			existingResourceQuotas: map[string]core_v1.ResourceQuotaSpec{
				"foo": {Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
			expectedActions: map[string]string{},
		},
	} {
		policyNodeList := &policyhierarchy_v1.PolicyNodeList{}
		for ns, rq := range testcase.policyNodeResourceQuotas {
			policyNodeList.Items = append(
				policyNodeList.Items,
				policyhierarchy_v1.PolicyNode{
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

		actions := syncer.computeResourceQuotaActionsWithResourceQuotaList(existingResourceQuotaList, policyNodeList)
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

type GetNamespaceEventActionTestCase struct {
	event             policynodewatcher.EventType
	noOperation       bool
	expectedOperation string
}

func TestSyncerGetEventNamespaceAction(t *testing.T) {
	syncer := newTestSyncer()

	namespaceName := "ns-name"
	policyNode := &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
	}

	for idx, testcase := range []GetNamespaceEventActionTestCase{
		{event: policynodewatcher.Added, expectedOperation: "create"},
		{event: policynodewatcher.Modified, noOperation: true},
		{event: policynodewatcher.Deleted, expectedOperation: "delete"},
	} {
		action := syncer.getEventNamespaceAction(testcase.event, policyNode)
		if testcase.noOperation {
			if action != nil {
				t.Errorf("Got unexpected non-nil action %#v for testcase %d, data %#v", action, idx, testcase)
			}
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

type GetResourceQuotaEventActionTestCase struct {
	event             policynodewatcher.EventType
	expectedOperation string
	noOperation       bool
	policyNode        policyhierarchy_v1.PolicyNode
}

func TestSyncerGetEventFesourceQuotaAction(t *testing.T) {
	syncer := newTestSyncer()

	namespaceName := "ns-name"
	syncer.client.Kubernetes().CoreV1().ResourceQuotas("ns-name").Create(&core_v1.ResourceQuota{
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

	for idx, testcase := range []GetResourceQuotaEventActionTestCase{
		{event: policynodewatcher.Added, expectedOperation: "create", policyNode: policyNodeWithSameRq},
		{event: policynodewatcher.Added, noOperation: true, policyNode: policyNodeWithoutRq},
		{event: policynodewatcher.Modified, expectedOperation: "delete", policyNode: policyNodeWithoutRq},
		{event: policynodewatcher.Modified, expectedOperation: "update", policyNode: policyNodeWithDifferentRq},
		{event: policynodewatcher.Modified, noOperation: true, policyNode: policyNodeWithSameRq},
		{event: policynodewatcher.Deleted, noOperation: true, policyNode: policyNodeWithoutRq},
	} {
		action := syncer.getEventResourceQuotaAction(testcase.event, &testcase.policyNode)
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
