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

// Package testing defines a common data-driven module testing framework.
package testing

import (
	"fmt"
	"testing"

	"github.com/google/nomos/pkg/syncer/policyhierarchycontroller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModuleEqualTestcase is a testcase for Module.Equal
type ModuleEqualTestcase struct {
	Name        string
	LHS         metav1.Object
	RHS         metav1.Object
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

// ModuleEqualTestcases is a list of testcases for testing a Module's Equal function.
type ModuleEqualTestcases []ModuleEqualTestcase

// RunAll runs all testcases in the list of ModuleEqualTestcases
func (tcs ModuleEqualTestcases) RunAll(module policyhierarchycontroller.Module, t *testing.T) {
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("ModuleEqualTestcase %s", tc.Name), tc.Run(module))
	}
}

// ModuleTest is a data-driven test for Modules that tests the non-boilerplate aspects of a module.
type ModuleTest struct {
	Module policyhierarchycontroller.Module
	Equals ModuleEqualTestcases
}

// RunAll runs all tests in the ModuleTest.
func (tc *ModuleTest) RunAll(t *testing.T) {
	tc.Equals.RunAll(tc.Module, t)
}
