package policynode

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type policyNodeEqualTestcase struct {
	name      string
	lhs       *v1.PolicyNode
	rhs       *v1.PolicyNode
	wantEqual bool
}

func (t *policyNodeEqualTestcase) Run(tt *testing.T) {
	equal := policyNodesEqual(t.lhs, t.rhs)
	if t.wantEqual == equal {
		return
	}

	diff := cmp.Diff(t.lhs, t.rhs, pnsIgnore...)
	if equal {
		tt.Errorf("wanted not equal, got equal: %s", diff)
	} else {
		tt.Errorf("wanted equal, got not equal: %s", diff)
	}
}

var time1 = time.Now()

var policyNodeEqualTestcases = []policyNodeEqualTestcase{
	{
		name: "basic",
		lhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{},
		},
		rhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{},
		},
		wantEqual: true,
	},
	{
		name: "different import token",
		lhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ImportToken: "1234567890123456789012345678901234567890",
			},
		},
		rhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ImportToken: "1234567890123456789012345678900000000000",
			},
		},
		wantEqual: true,
	},
	{
		name: "different import time",
		lhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ImportTime: metav1.NewTime(time1),
			},
		},
		rhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ImportTime: metav1.NewTime(time1.Add(time.Second)),
			},
		},
		wantEqual: true,
	},
	{
		name: "populated resource quota",
		lhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ResourceQuotaV1: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("5.0"),
							corev1.ResourceMemory:  resource.MustParse("5M"),
							corev1.ResourceStorage: resource.MustParse("5M"),
						},
					},
				},
			},
		},
		rhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ResourceQuotaV1: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("5.0"),
							corev1.ResourceMemory:  resource.MustParse("5M"),
							corev1.ResourceStorage: resource.MustParse("5M"),
						},
					},
				},
			},
		},
		wantEqual: true,
	},
	{
		name: "populated resource quota not equal",
		lhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ResourceQuotaV1: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5.0"),
						},
					},
				},
			},
		},
		rhs: &v1.PolicyNode{
			Spec: v1.PolicyNodeSpec{
				ResourceQuotaV1: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("6.0"),
						},
					},
				},
			},
		},
	},
}

func TestPolicyNodeEqual(t *testing.T) {
	for _, tc := range policyNodeEqualTestcases {
		t.Run(tc.name, tc.Run)
	}
}
