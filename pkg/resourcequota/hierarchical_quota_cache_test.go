package resourcequota

import (
	"strings"
	"testing"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"

	"github.com/google/nomos/pkg/testing/fakeinformers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type CacheTestCase struct {
	namespace              string
	newUsageList           corev1.ResourceList
	canAdmitExpected       bool
	expectedErrorSubstring string
}

func TestCanAdmit(t *testing.T) {

	// Limits and structure
	root := makeHierarchicalQuotaNode("kittiesandponies",
		corev1.ResourceList{
			"hay":  resource.MustParse("10"),
			"milk": resource.MustParse("5"),
		},
		true)

	root.Children = []v1alpha1.HierarchicalQuotaNode{
		*makeHierarchicalQuotaNode(
			"kitties",
			corev1.ResourceList{
				"hay": resource.MustParse("5"),
			},
			false),
		*makeHierarchicalQuotaNode(
			"ponies",
			corev1.ResourceList{
				"hay":  resource.MustParse("15"),
				"milk": resource.MustParse("5"),
			},
			false),
	}
	hierarchyQuota := []runtime.Object{
		makeHierarchicalQuota(root),
	}

	// Starting usages
	quotas := []runtime.Object{
		makeResourceQuota("kitties", corev1.ResourceList{
			"hay": resource.MustParse("2"),
		}),
		makeResourceQuota("ponies", corev1.ResourceList{
			"hay":  resource.MustParse("2"),
			"milk": resource.MustParse("2"),
		}),
	}

	resourceQuotaInformer := fakeinformers.NewResourceQuotaInformer(quotas...)
	hierarchicalQuotaInformer := fakeinformers.NewHierarchicalQuotaInformer(hierarchyQuota...)
	cache, err := NewHierarchicalQuotaCache(resourceQuotaInformer, hierarchicalQuotaInformer)
	if err != nil {
		t.Error(err)
		return
	}

	for i, tt := range []CacheTestCase{
		{ // Basic admit
			namespace: "kitties",
			newUsageList: corev1.ResourceList{
				"hay":  resource.MustParse("1"),
				"milk": resource.MustParse("1"),
			},
			canAdmitExpected: true,
		},
		{ // Admit no quota set
			namespace: "kitties",
			newUsageList: corev1.ResourceList{
				"bamboo": resource.MustParse("1"),
			},
			canAdmitExpected: true,
		},
		{ // violate at leaf but not at the policyspace
			namespace: "kitties",
			newUsageList: corev1.ResourceList{
				"hay": resource.MustParse("6"),
			},
			canAdmitExpected: true,
		},
		{ // violate at top (no limit at leaf)
			namespace: "kitties",
			newUsageList: corev1.ResourceList{
				"milk": resource.MustParse("7"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: milk",
		},
		{ // violate at top (higher limit at leaf)
			namespace: "ponies",
			newUsageList: corev1.ResourceList{
				"hay": resource.MustParse("12"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: hay",
		},
		{ // violate counting starting usage at leaf
			namespace: "ponies",
			newUsageList: corev1.ResourceList{
				"milk": resource.MustParse("4"),
			},
			canAdmitExpected:       false,
			expectedErrorSubstring: "policyspace kittiesandponies, requested: milk",
		},
		{ // violate counting starting usage at top (current = 2 + 2, limit at top = 10)
			namespace: "ponies",
			newUsageList: corev1.ResourceList{
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

func makeHierarchicalQuota(root *v1alpha1.HierarchicalQuotaNode) *v1alpha1.HierarchicalQuota {
	return &v1alpha1.HierarchicalQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "HierarchicalQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ResourceQuotaHierarchyName,
		},
		Spec: v1alpha1.HierarchicalQuotaSpec{
			Hierarchy: *root,
		},
	}
}

func makeHierarchicalQuotaNode(name string, limits corev1.ResourceList, abstract bool) *v1alpha1.HierarchicalQuotaNode {
	pnt := v1.HierarchyNodeNamespace
	if abstract {
		pnt = v1.HierarchyNodeAbstractNamespace
	}
	return &v1alpha1.HierarchicalQuotaNode{
		Name: name,
		Type: pnt,
		ResourceQuotaV1: &corev1.ResourceQuota{
			Spec: corev1.ResourceQuotaSpec{
				Hard: limits,
			},
		},
	}
}

func makeResourceQuota(namespace string, used corev1.ResourceList) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceQuotaObjectName,
			Namespace: namespace,
			Labels:    NomosQuotaLabels,
		},
		Status: corev1.ResourceQuotaStatus{
			Used: used,
		},
	}
}
