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
	"github.com/google/stolos/pkg/resource-quota"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
)

type ComputeResourceQuotaActionsTestCase struct {
	policyNodeResourceQuotas map[string]core_v1.ResourceQuotaSpec // Input policy node resource quota
	existingResourceQuotas   map[string]core_v1.ResourceQuotaSpec // Existing resource quotas in the cluster

	expectedActions map[string]string // A map of namespaces to the expected resource quota operation
}

func NewTestQuotaSyncer(quotas ...runtime.Object) *QuotaSyncer {
	informer := fakeinformers.NewResourceQuotaInformer(quotas...)
	return NewQuotaSyncer(fake.NewClient(), informer.Lister(), workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()))
}

type GetResourceQuotaEventActionTestCase struct {
	event             policynodewatcher.EventType
	expectedOperation string
	noOperation       bool
	policyNode        policyhierarchy_v1.PolicyNode
}

func TestSyncerGetEventFesourceQuotaAction(t *testing.T) {
	namespaceName := "ns-name"
	syncer := NewTestQuotaSyncer(&core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: resource_quota.ResourceQuotaObjectName, Labels: resource_quota.StolosQuotaLabels, Namespace: namespaceName,
		},
		Spec: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}},
	})

	policyNodeWithSameRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
			Policies: policyhierarchy_v1.Policies{
				ResourceQuota: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("42")}}},
		},
	}

	policyNodeWithDifferentRq := policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
			Policies: policyhierarchy_v1.Policies{
				ResourceQuota: core_v1.ResourceQuotaSpec{Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("43")}}},
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
		if action.Namespace() != namespaceName {
			t.Errorf("Added event should have name %s, got %s", namespaceName, action.Namespace())
		}
		if action.Operation() != testcase.expectedOperation {
			t.Errorf("Got unexpected operation %s for testcase %d, data %#v", action.Operation(), idx, testcase)
		}
	}
}
