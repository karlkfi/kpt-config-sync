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
	"github.com/google/stolos/pkg/util/set/stringset"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestSyncerComputeNamespaceActions(t *testing.T) {
	syncer := NewNamespaceSyncer(fake.NewClient())

	for _, testcase := range []ComputeNamespaceActionsTestCase{
		{ // Create terminating ns
			policyNodeNamespaces:  []string{"foo"},
			existingNamespaces:    []string{"bar"},
			terminatingNamespaces: []string{"foo"},
			needsCreate:           []string{"foo"},
			needsDelete:           []string{"bar"},
		},
		{ // need create, need delete
			policyNodeNamespaces:  []string{"foo", "foo2"},
			existingNamespaces:    []string{"bar", "bar2"},
			terminatingNamespaces: []string{"baz"},
			needsCreate:           []string{"foo", "foo2"},
			needsDelete:           []string{"bar", "bar2"},
		},
		{ // need create
			policyNodeNamespaces:  []string{"foo", "bar"},
			existingNamespaces:    []string{"bar"},
			terminatingNamespaces: []string{},
			needsCreate:           []string{"foo"},
			needsDelete:           []string{},
		},
		{ // need delete
			policyNodeNamespaces:  []string{"foo"},
			existingNamespaces:    []string{"bar", "foo"},
			terminatingNamespaces: []string{"baz"},
			needsCreate:           []string{},
			needsDelete:           []string{"bar"},
		},
		{ // No diff
			policyNodeNamespaces:  []string{"foo"},
			existingNamespaces:    []string{"foo"},
			terminatingNamespaces: []string{"baz"},
			needsCreate:           []string{},
			needsDelete:           []string{},
		},
	} {
		policyNodes := []*policyhierarchy_v1.PolicyNode{}
		for _, value := range testcase.policyNodeNamespaces {
			policyNodes = append(
				policyNodes, &policyhierarchy_v1.PolicyNode{ObjectMeta: meta_v1.ObjectMeta{Name: value}})
		}

		namespaceList := &core_v1.NamespaceList{}
		for _, value := range testcase.existingNamespaces {
			namespaceList.Items = append(namespaceList.Items, createNamespace(value, core_v1.NamespaceActive))
		}
		for _, value := range testcase.terminatingNamespaces {
			namespaceList.Items = append(namespaceList.Items, createNamespace(value, core_v1.NamespaceTerminating))
		}

		actions := syncer.computeActions(namespaceList, policyNodes)

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
