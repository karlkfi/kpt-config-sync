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

package syncer

import (
	"reflect"
	"testing"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta/fake"
	"github.com/google/stolos/pkg/resourcequota"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
)

type GetResourceQuotaEventActionTestCase struct {
	expectedOperation string
	expectedQuota     core_v1.ResourceList
	noOperation       bool
	policyNode        policyhierarchy_v1.PolicyNode
}

var policyNodes = []runtime.Object{
	makePolicyNode("top", "", core_v1.ResourceList{"configmaps": resource.MustParse("10")}, true),
	makePolicyNode("mid", "top", core_v1.ResourceList{"memory": resource.MustParse("5")}, true),
	makePolicyNode("child1", "mid", core_v1.ResourceList{}, false),
	makePolicyNode("child2", "mid", core_v1.ResourceList{}, false),
}

func TestSyncerGetEventResourceQuotaAction(t *testing.T) {
	namespaceName := "ns-name"
	informer := fakeinformers.NewResourceQuotaInformer(&core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: resourcequota.ResourceQuotaObjectName, Labels: resourcequota.StolosQuotaLabels, Namespace: namespaceName,
		},
		Spec: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}},
	})
	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	syncer := NewQuotaSyncer(fake.NewClient(), informer, policyNodeInformer,
		workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()))

	policyNodeWithSameRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
			Policies: policyhierarchy_v1.Policies{
				ResourceQuotaV1: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
		},
	}

	policyNodeWithDifferentRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
			Policies: policyhierarchy_v1.Policies{
				ResourceQuotaV1: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}},
		},
	}
	policyNodeWithoutRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
		},
	}

	policyNodeWithGap := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
			Parent:      "mid",
			Policies: policyhierarchy_v1.Policies{
				ResourceQuotaV1: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{"memory": resource.MustParse("2")}}},
		},
	}

	policyNodeForPolicypace := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "mid",
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: true,
			Parent:      "top",
			Policies: policyhierarchy_v1.Policies{
				ResourceQuotaV1: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{"memory": resource.MustParse("2")}}},
		},
	}

	expectedGapQuotaLimits := core_v1.ResourceList{
		"memory":     resource.MustParse("2"),
		"configmaps": resource.MustParse("10")}

	for idx, testcase := range []GetResourceQuotaEventActionTestCase{
		{expectedOperation: "delete", policyNode: policyNodeWithoutRq},
		{expectedOperation: "upsert", policyNode: policyNodeWithDifferentRq},
		{expectedOperation: "upsert", policyNode: policyNodeWithSameRq},
		{expectedOperation: "upsert", expectedQuota: expectedGapQuotaLimits, policyNode: policyNodeWithGap},
		{noOperation: true, policyNode: policyNodeForPolicypace},
	} {
		action := syncer.getUpdateAction(&testcase.policyNode)
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
		if action.Namespace() != namespaceName {
			t.Errorf("Added event should have name %s, got %s", namespaceName, action.Namespace())
		}
		if action.Operation() != testcase.expectedOperation {
			t.Errorf("Got unexpected operation %s for testcase %d, data %#v", action.Operation(), idx, testcase)
		}
		if testcase.expectedQuota != nil && !reflect.DeepEqual(action.ResourceQuotaSpec().Hard, testcase.expectedQuota) {
			t.Errorf("Got unexpected resource quota %#v for testcase %d, expected %#v", action.ResourceQuotaSpec().Hard, idx, testcase.expectedQuota)
		}
	}
}

func TestFillResourceQuotaLeafGaps(t *testing.T) {
	quotas := []runtime.Object{
		&core_v1.ResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: resourcequota.ResourceQuotaObjectName, Labels: resourcequota.StolosQuotaLabels, Namespace: "child1",
			},
			Spec: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{"cpu": resource.MustParse("2")}},
		},
		&core_v1.ResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: resourcequota.ResourceQuotaObjectName, Labels: resourcequota.StolosQuotaLabels, Namespace: "child2",
			},
			Spec: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{"configmaps": resource.MustParse("3")}},
		}}

	informer := fakeinformers.NewResourceQuotaInformer(quotas...)
	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	syncer := NewQuotaSyncer(fake.NewClient(), informer, policyNodeInformer,
		workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()))
	policyNodeItems, _ := policyNodeInformer.Lister().List(labels.Everything())
	actions, _ := syncer.fillResourceQuotaLeafGaps(policyNodeItems)

	if len(actions) != 2 {
		t.Errorf("Expected 2 actions, one for each child, but got %d", len(actions))
	}

	child1ExpectedSpec := core_v1.ResourceList{"cpu": resource.MustParse("2"), "configmaps": resource.MustParse("10"), "memory": resource.MustParse("5")}
	child2ExpectedSpec := core_v1.ResourceList{"configmaps": resource.MustParse("3"), "memory": resource.MustParse("5")}
	var child1Actual, child2Actual core_v1.ResourceList
	for _, actual := range actions {
		if actual.Namespace() == "child1" {
			child1Actual = actual.ResourceQuotaSpec().Hard
		}
		if actual.Namespace() == "child2" {
			child2Actual = actual.ResourceQuotaSpec().Hard
		}
	}
	if !reflect.DeepEqual(child1Actual, child1ExpectedSpec) {
		t.Errorf("Expected child1 actions %v, but got %v", child1ExpectedSpec, child1Actual)
	}

	if !reflect.DeepEqual(child2Actual, child2ExpectedSpec) {
		t.Errorf("Expected child1 actions %v, but got %v", child2ExpectedSpec, child2Actual)
	}
}

func makePolicyNode(name string, parent string, limits core_v1.ResourceList, policyspace bool) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: policyspace,
			Parent:      parent,
			Policies: policyhierarchy_v1.Policies{
				ResourceQuotaV1: core_v1.ResourceQuotaSpec{
					Hard: limits,
				},
			},
		},
	}
}
