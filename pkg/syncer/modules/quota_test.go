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

	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/syncer/hierarchy"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	test "github.com/google/nomos/pkg/syncer/policyhierarchycontroller/testing"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func WithResourceQuota(hardResList core_v1.ResourceList) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			ResourceQuotaV1: &core_v1.ResourceQuota{
				Spec: core_v1.ResourceQuotaSpec{
					Hard: hardResList,
				},
			},
		},
	}
}

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
		Aggregation: test.ModuleAggregationTestcases{
			test.ModuleAggregationTestcase{
				Name: "Base case",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithResourceQuota(core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")}),
				},
				Expect: hierarchy.Instances{
					&core_v1.ResourceQuota{
						ObjectMeta: meta_v1.ObjectMeta{
							Name:   resourcequota.ResourceQuotaObjectName,
							Labels: resourcequota.NewNomosQuotaLabels(),
						},
						Spec: core_v1.ResourceQuotaSpec{
							Hard: core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")},
						},
					},
				},
			},
			test.ModuleAggregationTestcase{
				Name: "Accumulate extra fields",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithResourceQuota(core_v1.ResourceList{core_v1.ResourceMemory: resource.MustParse("100")}),
					WithResourceQuota(core_v1.ResourceList{core_v1.ResourceCPU: resource.MustParse("10")}),
				},
				Expect: hierarchy.Instances{
					&core_v1.ResourceQuota{
						ObjectMeta: meta_v1.ObjectMeta{
							Name:   resourcequota.ResourceQuotaObjectName,
							Labels: resourcequota.NewNomosQuotaLabels(),
						},
						Spec: core_v1.ResourceQuotaSpec{
							Hard: core_v1.ResourceList{
								core_v1.ResourceCPU:    resource.MustParse("10"),
								core_v1.ResourceMemory: resource.MustParse("100"),
							},
						},
					},
				},
			},
			test.ModuleAggregationTestcase{
				Name: "Accumulate min value",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithResourceQuota(core_v1.ResourceList{core_v1.ResourceMemory: resource.MustParse("100")}),
					WithResourceQuota(core_v1.ResourceList{core_v1.ResourceMemory: resource.MustParse("10")}),
				},
				Expect: hierarchy.Instances{
					&core_v1.ResourceQuota{
						ObjectMeta: meta_v1.ObjectMeta{
							Name:   resourcequota.ResourceQuotaObjectName,
							Labels: resourcequota.NewNomosQuotaLabels(),
						},
						Spec: core_v1.ResourceQuotaSpec{
							Hard: core_v1.ResourceList{
								core_v1.ResourceMemory: resource.MustParse("10"),
							},
						},
					},
				},
			},
			test.ModuleAggregationTestcase{
				Name: "Accumulate nil quota",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithResourceQuota(core_v1.ResourceList{core_v1.ResourceMemory: resource.MustParse("100")}),
					WithResourceQuota(core_v1.ResourceList{}),
				},
				Expect: hierarchy.Instances{
					&core_v1.ResourceQuota{
						ObjectMeta: meta_v1.ObjectMeta{
							Name:   resourcequota.ResourceQuotaObjectName,
							Labels: resourcequota.NewNomosQuotaLabels(),
						},
						Spec: core_v1.ResourceQuotaSpec{
							Hard: core_v1.ResourceList{
								core_v1.ResourceMemory: resource.MustParse("100"),
							},
						},
					},
				},
			},
			test.ModuleAggregationTestcase{
				Name: "No quota",
				PolicyNodes: []*policyhierarchy_v1.PolicyNode{
					WithResourceQuota(core_v1.ResourceList{}),
				},
				Expect: hierarchy.Instances{},
			},
		},
	}
	testSuite.RunAll(t)
}
