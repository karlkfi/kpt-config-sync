// Reviewed by sunilarora
package resourcequota

import (
	"strings"
	"testing"

	pn_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		}, true),
		makePolicyNode("kitties", "kittiesandponies", core_v1.ResourceList{
			"hay": resource.MustParse("5"),
		}, false),
		makePolicyNode("ponies", "kittiesandponies", core_v1.ResourceList{
			"hay":  resource.MustParse("15"),
			"milk": resource.MustParse("5"),
		}, false),
	}

	// Starting usages
	quotas := []runtime.Object{
		makeResourceQuota("kitties", core_v1.ResourceList{
			"hay": resource.MustParse("2"),
		}),
		makeResourceQuota("ponies", core_v1.ResourceList{
			"hay":  resource.MustParse("2"),
			"milk": resource.MustParse("2"),
		}),
	}

	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	resourceQuotaInformer := fakeinformers.NewResourceQuotaInformer(quotas...)
	cache, err := NewHierarchicalQuotaCache(policyNodeInformer, resourceQuotaInformer)
	if err != nil {
		t.Error(err)
		return
	}

	for i, tt := range []CacheTestCase{
		{ // Basic admit
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"hay":  resource.MustParse("1"),
				"milk": resource.MustParse("1"),
			},
			canAdmitExpected: true,
		},
		{ // Admit no quota set
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"bamboo": resource.MustParse("1"),
			},
			canAdmitExpected: true,
		},
		{ // violate at leaf but not at the policyspace
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"hay": resource.MustParse("6"),
			},
			canAdmitExpected: true,
		},
		{ // violate at top (no limit at leaf)
			namespace: "kitties",
			newUsageList: core_v1.ResourceList{
				"milk": resource.MustParse("7"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: milk",
		},
		{ // violate at top (higher limit at leaf)
			namespace: "ponies",
			newUsageList: core_v1.ResourceList{
				"hay": resource.MustParse("12"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: hay",
		},
		{ // violate counting starting usage at leaf
			namespace: "ponies",
			newUsageList: core_v1.ResourceList{
				"milk": resource.MustParse("4"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: milk",
		},
		{ // violate counting starting usage at top (current = 2 + 2, limit at top = 10)
			namespace: "ponies",
			newUsageList: core_v1.ResourceList{
				"hay": resource.MustParse("7"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: hay",
		},
	} {
		err := cache.Admit(tt.namespace, tt.newUsageList)
		if (err == nil) != tt.canAdmitExpected {
			t.Errorf("Expected %t but got %t admitting test case [%d]", tt.canAdmitExpected, err == nil, i)
		}
		if err != nil && len(tt.expectedErrorSubstring) > 0 {
			if !strings.Contains(err.Error(), tt.expectedErrorSubstring) {
				t.Errorf("Expected error [%s] to contain substring [%s]", err, tt.expectedErrorSubstring)
			}
		}
	}
}

func TestUpdateLeaf(t *testing.T) {

	// Limits and structure
	policyNodes := []runtime.Object{
		makePolicyNode("kittiesandponies", "", core_v1.ResourceList{
			"hay":  resource.MustParse("10"),
			"milk": resource.MustParse("5"),
		}, true),
		makePolicyNode("kitties", "kittiesandponies", core_v1.ResourceList{
			"hay": resource.MustParse("5"),
		}, false),
		makePolicyNode("ponies", "kittiesandponies", core_v1.ResourceList{
			"hay":  resource.MustParse("15"),
			"milk": resource.MustParse("5"),
		}, false),
	}

	// Starting usages
	quotas := []runtime.Object{
		makeResourceQuota("kitties", core_v1.ResourceList{
			"hay": resource.MustParse("2"),
		}),
		makeResourceQuota("ponies", core_v1.ResourceList{
			"hay":  resource.MustParse("2"),
			"milk": resource.MustParse("2"),
		}),
	}

	policyNodeInformer := fakeinformers.NewPolicyNodeInformer(policyNodes...)
	resourceQuotaInformer := fakeinformers.NewResourceQuotaInformer(quotas...)
	cache, err := NewHierarchicalQuotaCache(policyNodeInformer, resourceQuotaInformer)
	if err != nil {
		t.Error(err)
		return
	}

	// Remove milk and change hay to 3.
	namespaces, err := cache.UpdateLeaf(*makeResourceQuota("ponies", core_v1.ResourceList{
		"hay": resource.MustParse("3"),
	}))

	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}

	if len(namespaces) != 1 || namespaces[0] != "kittiesandponies" {
		t.Errorf("Unexpected namespaces to updated %s", namespaces)
	}

	expectedNewUsage := core_v1.ResourceList{
		"hay":  resource.MustParse("5"),
		"milk": resource.MustParse("0"),
	}

	if !resourceListEqual(cache.quotas["kittiesandponies"].quota.Status.Used, expectedNewUsage) {
		t.Errorf("Unexpected new usage %#v", cache.quotas["kittiesandponies"].quota.Status.Used)
	}
}

func makePolicyNode(name string, parent string, limits core_v1.ResourceList, policyspace bool) *pn_v1.PolicyNode {
	pnt := pn_v1.Namespace
	if policyspace {
		pnt = pn_v1.Policyspace
	}
	return &pn_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: pn_v1.PolicyNodeSpec{
			Parent: parent,
			Type:   pnt,
			Policies: pn_v1.Policies{
				ResourceQuotaV1: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: limits,
					},
				},
			},
		},
	}
}

func makeResourceQuota(namespace string, used core_v1.ResourceList) *core_v1.ResourceQuota {
	return &core_v1.ResourceQuota{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      ResourceQuotaObjectName,
			Namespace: namespace,
			Labels:    NomosQuotaLabels,
		},
		Status: core_v1.ResourceQuotaStatus{
			Used: used,
		},
	}
}
