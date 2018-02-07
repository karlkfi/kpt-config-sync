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

package remotecluster

import (
	"testing"
	"time"

	"reflect"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/informers/externalversions"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta/fake"
	typed_v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var existingLocalNodes = []*policyhierarchy_v1.PolicyNode{
	newPolicyNode("acme", "", "2", true),
	newPolicyNode("eng", "acme", "1", true),
	newPolicyNode("frontend", "eng", "1", false),
}

func setUpLocalPolicyNodes(t *testing.T) (listers_v1.PolicyNodeLister, typed_v1.PolicyNodeInterface) {
	client := fake.NewClient()
	policyNodeInterface := client.PolicyHierarchy().StolosV1().PolicyNodes()

	for _, n := range existingLocalNodes {
		_, err := policyNodeInterface.Create(n)
		if err != nil {
			t.Fatalf("Failed to set up local policy node: %v", err)
		}
	}

	informerFactory := externalversions.NewSharedInformerFactory(
		client.PolicyHierarchy(), time.Minute)
	policyNodeLister := informerFactory.Stolos().V1().PolicyNodes().Lister()
	informerFactory.Start(nil)
	informerFactory.WaitForCacheSync(nil)

	return policyNodeLister, policyNodeInterface
}

func newPolicyNode(name string, parent string, cpuLimit string, policyspace bool) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
			// Fake client doesn't populate resourceVersion. Roll our own.
			Annotations: map[string]string{"resourceVersion": time.Now().String()},
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: policyspace,
			Parent:      parent,
			Policies: policyhierarchy_v1.Policies{
				ResourceQuota: core_v1.ResourceQuotaSpec{
					Hard: core_v1.ResourceList{"cpu": resource.MustParse(cpuLimit)},
				},
			},
		},
	}
}

type upsertTestCase struct {
	testName     string
	node         *policyhierarchy_v1.PolicyNode
	expectUpdate bool
}

var upsertTestCases = []upsertTestCase{
	{
		testName:     "Create new node",
		node:         newPolicyNode("new", "acme", "1", false),
		expectUpdate: false,
	},
	{
		testName:     "No update needed",
		node:         newPolicyNode("eng", "acme", "1", false),
		expectUpdate: false,
	},
	{
		testName:     "Update needed",
		node:         newPolicyNode("eng", "acme", "2", false),
		expectUpdate: true,
	},
}

func TestPolicyNodeUpsertAction(t *testing.T) {

	for _, testCase := range upsertTestCases {
		t.Run(testCase.testName, func(t *testing.T) {
			policyNodeLister, policyNodeInterface := setUpLocalPolicyNodes(t)

			if testCase.expectUpdate {
				_, err := policyNodeInterface.Get(testCase.node.Name, meta_v1.GetOptions{})
				if err != nil {
					t.Fatalf("Failed to get existing policynode: %v", err)
				}
			}

			action := NewPolicyNodeUpsertAction(testCase.node, policyNodeLister, policyNodeInterface)
			err := action.Execute()
			if err != nil {
				t.Fatalf("Failed to execute action %s", action)
			}

			n, err := policyNodeInterface.Get(testCase.node.Name, meta_v1.GetOptions{})
			if err != nil {
				t.Fatalf("Failed to upsert policynode: %v", err)
			}

			if !reflect.DeepEqual(testCase.node.Spec, n.Spec) {
				t.Fatalf("Spec doesn't match")
			}

			if testCase.expectUpdate {
				// canonicalCopy discards annotations.
				if n.Annotations["resourceVersion"] != "" {
					t.Fatalf("Expected policynode to be updated")
				}
			}
		})
	}
}

type deleteTestCase struct {
	testName string
	node     *policyhierarchy_v1.PolicyNode
}

var deleteTestCases = []deleteTestCase{
	{
		testName: "Delete non-existent node",
		node:     newPolicyNode("new", "acme", "1", false),
	},
	{
		testName: "Delete existing node",
		node:     newPolicyNode("eng", "acme", "1", true),
	},
}

func TestPolicyNodeDeleteAction(t *testing.T) {

	for _, tc := range deleteTestCases {
		t.Run(tc.testName, func(t *testing.T) {
			policyNodeLister, policyNodeInterface := setUpLocalPolicyNodes(t)

			action := NewPolicyNodeDeleteAction(tc.node, policyNodeLister, policyNodeInterface)
			err := action.Execute()
			if err != nil {
				t.Fatalf("Unexpected error when deleting: %v", err)
			}

			_, err = policyNodeInterface.Get(tc.node.Name, meta_v1.GetOptions{})
			if err == nil {
				t.Fatalf("Failed to delete policyspace")
			}
			if !api_errors.IsNotFound(err) {
				t.Fatalf("Expected error: %v", err)
			}
		})
	}
}
