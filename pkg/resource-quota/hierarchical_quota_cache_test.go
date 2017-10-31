package resource_quota

import (
	"testing"
	"strings"

	pn_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type CacheTestCase struct {
	namespace              string
	newUsageList           core_v1.ResourceList
	canAdmitExpected       bool
	expectedErrorSubstring string
}

func TestCanAdmit(t *testing.T) {
	// Limits and structure
	policyNodes := []runtime.Object{
		makePolicyNode("kittiesandponies", "", core_v1.ResourceList{
			"hay":  resource.MustParse("10"),
			"milk": resource.MustParse("5"),
		}, ),
		makePolicyNode("kitties", "kittiesandponies", core_v1.ResourceList{
			"hay": resource.MustParse("5"),
		}, ),
		makePolicyNode("ponies", "kittiesandponies", core_v1.ResourceList{
			"hay":  resource.MustParse("15"),
			"milk": resource.MustParse("5"),
		}, ),
	}

	// Starting usages
	quotas := []runtime.Object{
		&core_v1.ResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      ResourceQuotaObjectName,
				Namespace: "kitties",
			},
			Status: core_v1.ResourceQuotaStatus{
				Used: core_v1.ResourceList{
					"hay": resource.MustParse("2"),
				},
			},
		},
		&core_v1.ResourceQuota{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      ResourceQuotaObjectName,
				Namespace: "ponies",
			},
			Status: core_v1.ResourceQuotaStatus{
				Used: core_v1.ResourceList{
					"hay":  resource.MustParse("2"),
					"milk": resource.MustParse("2"),
				},
			},
		},
	}

	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	resourceQuotaInformer := fakeinformers.NewResourceQuotaInformer(quotas...)
	cache, err := NewHierarchicalQuotaCache(policyNodeInformer, resourceQuotaInformer)
	if err != nil {
		t.Error(err)
		return
	}

	for i, tt := range []CacheTestCase{
		{// Basic admit
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"hay":  resource.MustParse("1"),
				"milk": resource.MustParse("1"),
			},
			canAdmitExpected: true,
		},
		{// Admit no quota set
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"bamboo": resource.MustParse("1"),
			},
			canAdmitExpected: true,
		},
		{// violate at leaf
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"hay": resource.MustParse("7"),
			},
			canAdmitExpected: false,
			expectedErrorSubstring: "namespace kitties, requested: hay",
		},
		{// violate at top (no limit at leaf)
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"milk": resource.MustParse("7"),
			},
			canAdmitExpected: false,
			expectedErrorSubstring: "namespace kittiesandponies, requested: milk",
		},
		{// violate at top (higher limit at leaf)
			namespace: "ponies",
			newUsageList: core_v1.ResourceList{
				"hay": resource.MustParse("12"),
			},
			canAdmitExpected: false,
			expectedErrorSubstring: "namespace kittiesandponies, requested: hay",
		},
		{// violate counting starting usage at leaf
			namespace: "ponies",
			newUsageList: core_v1.ResourceList{
				"milk": resource.MustParse("4"),
			},
			canAdmitExpected: false,
			expectedErrorSubstring: "namespace ponies, requested: milk",
		},
		{// violate counting starting usage at top (current = 2 + 2, limit at top = 10)
			namespace: "ponies",
			newUsageList: core_v1.ResourceList{
				"hay": resource.MustParse("7"),
			},
			canAdmitExpected: false,
			expectedErrorSubstring: "namespace kittiesandponies, requested: hay",
		},
	} {
		err := cache.Admit(tt.namespace, tt.newUsageList)
		if (err == nil) != tt.canAdmitExpected {
			t.Errorf("Expected %s but got %s admitting test case [%d]", tt.canAdmitExpected, err == nil, i)
		}
		if err != nil && len(tt.expectedErrorSubstring) > 0 {
			if !strings.Contains(err.Error(), tt.expectedErrorSubstring) {
				t.Errorf("Expected error [%s] to contain substring [%s]", err, tt.expectedErrorSubstring)
			}
		}
	}
}

func makePolicyNode(name string, parent string, limits core_v1.ResourceList) *pn_v1.PolicyNode {
	return &pn_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: pn_v1.PolicyNodeSpec{
			Parent: parent,
			Policies: pn_v1.Policies{
				ResourceQuota: core_v1.ResourceQuotaSpec{
					Hard: limits,
				},
			},
		},
	}
}
