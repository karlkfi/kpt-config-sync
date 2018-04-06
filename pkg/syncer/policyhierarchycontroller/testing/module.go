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

package testing

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/nomos/pkg/syncer/hierarchy"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModuleEqualTestcase is a testcase for module equals.
type ModuleEqualTestcase struct {
	Name        string
	LHS         meta_v1.Object
	RHS         meta_v1.Object
	ExpectEqual bool
}

// Run runs the testcase.
func (tc ModuleEqualTestcase) Run(module policyhierarchycontroller.Module) func(t *testing.T) {
	return func(t *testing.T) {
		result := module.Equal(tc.LHS, tc.RHS)
		if result != tc.ExpectEqual {
			t.Errorf("expected %v got %v", tc.ExpectEqual, result)
		}
	}
}

type ModuleEqualTestcases []ModuleEqualTestcase

func (tcs ModuleEqualTestcases) RunAll(module policyhierarchycontroller.Module, t *testing.T) {
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("ModuleEqualTestcase %s", tc.Name), tc.Run(module))
	}
}

type ModuleAggregationTestcase struct {
	Name       string
	Aggregated hierarchy.AggregatedNode
	PolicyNode *policyhierarchy_v1.PolicyNode
	Expect     hierarchy.Instances
}

// Run runs the testcase.
func (tc ModuleAggregationTestcase) Run(module policyhierarchycontroller.Module) func(t *testing.T) {
	return func(t *testing.T) {
		actual := tc.Aggregated.Aggregated(tc.PolicyNode).Generate()

		actual.Sort()
		tc.Expect.Sort()

		if len(tc.Expect) == 0 && len(actual) == 0 {
			return
		}

		for i := 0; i < len(tc.Expect); i++ {
			expect := tc.Expect[i]
			act := actual[i]
			if !module.Equal(expect, act) {
				cfg := spew.NewDefaultConfig()
				cfg.Indent = "  "
				t.Errorf(cfg.Sprintf("index %v expected:\n%v\ngot\n%v", i, tc.Expect[i], actual[i]))
			}
		}
	}
}

type ModuleAggregationTestcases []ModuleAggregationTestcase

func (tcs ModuleAggregationTestcases) RunAll(module policyhierarchycontroller.Module, t *testing.T) {
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("ModuleAggregationTestcase %s", tc.Name), tc.Run(module))
	}
}

type ModuleTest struct {
	Module      policyhierarchycontroller.Module
	Equals      ModuleEqualTestcases
	Aggregation ModuleAggregationTestcases
}

func (tc *ModuleTest) RunAll(t *testing.T) {
	tc.Equals.RunAll(tc.Module, t)
	tc.Aggregation.RunAll(tc.Module, t)
}
