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
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestQuota(t *testing.T) {
	testSuite := test.ModuleTest{
		Module: NewResourceQuota(nil, nil),
		Equals: test.ModuleEqualTestcases{
			test.ModuleEqualTestcase{
				Name:        "Both empty",
				LHS:         &core_v1.ResourceQuota{},
				RHS:         &core_v1.ResourceQuota{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Ignores status",
				LHS:  &core_v1.ResourceQuota{},
				RHS: &core_v1.ResourceQuota{
					Status: core_v1.ResourceQuotaStatus{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
					},
				},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Nil / non nil empty maps equivalent",
				LHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{},
				},
				RHS:         &core_v1.ResourceQuota{},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Equal Limits",
				LHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
					},
				},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Equal Limits Int vs Float",
				LHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10.0")},
					},
				},
				ExpectEqual: true,
			},
			test.ModuleEqualTestcase{
				Name: "Not equal Limits",
				LHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("11")},
					},
				},
				ExpectEqual: false,
			},
			test.ModuleEqualTestcase{
				Name: "Different keys",
				LHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
					},
				},
				RHS: &core_v1.ResourceQuota{
					Spec: core_v1.ResourceQuotaSpec{
						Hard: core_v1.ResourceList{core_v1.ResourceMemory: resource.MustParse("11")},
					},
				},
				ExpectEqual: false,
			},
		},
	}
	testSuite.RunAll(t)
}
