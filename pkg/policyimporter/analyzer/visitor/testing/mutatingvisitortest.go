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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"k8s.io/apimachinery/pkg/api/resource"
)

// MutatingVisitorTestcase is a struct that halps for testing
// MutatingVisitor types.
type MutatingVisitorTestcase struct {
	Name         string
	Input        *ast.Context
	ExpectOutput *ast.Context
	ExpectErr    bool
}

// ResourceVersionCmp provides a comparer option for resource.Quantity
func ResourceVersionCmp() cmp.Option {
	return cmp.Comparer(func(lhs, rhs resource.Quantity) bool {
		return lhs.Cmp(rhs) == 0
	})
}

// Runf returns a function that runs the testcase.
func (tc *MutatingVisitorTestcase) Runf(visitor ast.CheckingVisitor) func(t *testing.T) {
	return func(t *testing.T) {
		output := tc.Input.Accept(visitor)
		actual, ok := output.(*ast.Context)
		if !ok {
			t.Fatalf("Wrong type returned %#v", output)
		}
		err := visitor.Result()
		if (err != nil) != tc.ExpectErr {
			if tc.ExpectErr {
				t.Fatalf("expected error, got nil")
			} else {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		if tc.ExpectErr {
			return
		}
		if !cmp.Equal(actual, tc.ExpectOutput, ResourceVersionCmp()) {
			t.Fatalf("mismatch on expected vs actual: %s", cmp.Diff(tc.ExpectOutput, actual, ResourceVersionCmp()))
		}
	}
}

// MutatingVisitorTestcases specifies a list of testcases for the
type MutatingVisitorTestcases struct {
	VisitorCtor func() ast.CheckingVisitor
	Testcases   []MutatingVisitorTestcase
}

// Run runs all testcases.
func (tcs *MutatingVisitorTestcases) Run(t *testing.T) {
	for _, testcase := range tcs.Testcases {
		t.Run(testcase.Name, testcase.Runf(tcs.VisitorCtor()))
	}
}
