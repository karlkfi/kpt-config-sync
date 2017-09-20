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

	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policynodewatcher"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/util/set/stringset"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComputeActionsTestCase struct {
	policyNodeNamespaces  []string // namespaces defined in the poicy node objects
	existingNamespaces    []string // namespaces in the active state
	terminatingNamespaces []string // namespaces in the "terminating" state

	expectError bool     // True if we expect the function to error out
	needsCreate []string // namespaces that will be deleted
	needsDelete []string // namespaces that will be created
}

func createNamespace(name string, phase core_v1.NamespacePhase) core_v1.Namespace {
	return core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{Name: name},
		Status:     core_v1.NamespaceStatus{Phase: phase},
	}
}

func TestSyncerComputeActions(t *testing.T) {
	syncer := &Syncer{}

	for idx, testcase := range []ComputeActionsTestCase{
		{ // encounter error
			policyNodeNamespaces:  []string{"foo"},
			existingNamespaces:    []string{"bar"},
			terminatingNamespaces: []string{"foo"},
			expectError:           true,
			needsCreate:           []string{},
			needsDelete:           []string{},
		},
		{ // need create, need delete
			policyNodeNamespaces:  []string{"foo", "foo2"},
			existingNamespaces:    []string{"bar", "bar2"},
			terminatingNamespaces: []string{"baz"},
			expectError:           false,
			needsCreate:           []string{"foo", "foo2"},
			needsDelete:           []string{"bar", "bar2"},
		},
		{ // need create
			policyNodeNamespaces:  []string{"foo", "bar"},
			existingNamespaces:    []string{"bar"},
			terminatingNamespaces: []string{},
			expectError:           false,
			needsCreate:           []string{"foo"},
			needsDelete:           []string{},
		},
		{ // need delete
			policyNodeNamespaces:  []string{"foo"},
			existingNamespaces:    []string{"bar", "foo"},
			terminatingNamespaces: []string{"baz"},
			expectError:           false,
			needsCreate:           []string{},
			needsDelete:           []string{"bar"},
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

		actions, err := syncer.computeActions(namespaceList, policyNodeList)
		if testcase.expectError {
			if err != nil {
				t.Errorf("Testcase %d failed to produce error.  %#v", idx, testcase)
			}
			continue
		}

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

	}
}

type GetEventActionTestCase struct {
	event             policynodewatcher.EventType
	noOperation       bool
	expectedOperation string
}

func TestSyncerGetEventAction(t *testing.T) {
	syncer := &Syncer{}

	namespaceName := "ns-name"
	policyNode := &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
	}

	for idx, testcase := range []GetEventActionTestCase{
		{event: policynodewatcher.Added, expectedOperation: "create"},
		{event: policynodewatcher.Modified, noOperation: true},
		{event: policynodewatcher.Deleted, expectedOperation: "delete"},
	} {
		action := syncer.getEventAction(testcase.event, policyNode)
		if testcase.noOperation {
			if action != nil {
				t.Errorf("Got unexpected non-nil action %#v for testcase %d, data %#v", action, idx, testcase)
			}
			continue
		}
		if action.Name() != namespaceName {
			t.Errorf("Added event should have name %s, got %s", namespaceName)
		}
		if action.Operation() != testcase.expectedOperation {
			t.Errorf("Got unexpected operation %s for testcase %d, data %#v", action.Operation(), idx, testcase)
		}
	}

}
