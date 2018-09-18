/*
Copyright 2018 The Nomos Authors.
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
// Reviewed by sunilarora

package modules

import (
	"testing"

	test "github.com/google/nomos/pkg/syncer/policyhierarchycontroller/testing"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestQuota(t *testing.T) {
	testSuite := test.ModuleTest{
		Module: NewResourceQuota(nil, nil),
		Equals: test.ModuleEqualTestcases{
			test.ModuleEqualTestcase{
				Name:        "Both empty",
				LHS:         &corev1.ResourceQuota{},
				RHS:         &corev1.ResourceQuota{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Ignores status",
				LHS:  &corev1.ResourceQuota{},
				RHS: &corev1.ResourceQuota{
					Status: corev1.ResourceQuotaStatus{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")},
					},
				},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Nil / non nil empty maps equivalent",
				LHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{},
				},
				RHS:         &corev1.ResourceQuota{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Equal Limits",
				LHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")},
					},
				},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Equal Limits Int vs Float",
				LHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10.0")},
					},
				},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Not equal Limits",
				LHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("11")},
					},
				},
				ExpectEqual: false,
			},
			test.ModuleEqualTestcase{
				Name: "Different keys",
				LHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &corev1.ResourceQuota{
					Spec: corev1.ResourceQuotaSpec{
						Hard: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("11")},
					},
				},
				ExpectEqual: false,
			},
		},
	}
	testSuite.RunAll(t)
}
