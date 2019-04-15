/*
/*
Copyright 2018 The CSP Config Management Authors.

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
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	visitorpkg "github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/util/discovery"
)

// MutatingVisitorTestcase is a struct that halps for testing
// MutatingVisitor types.
type MutatingVisitorTestcase struct {
	Name         string
	Input        *ast.Root
	ExpectOutput *ast.Root
	ExpectErr    bool
	ExpectNoop   bool // Output is expected to the exact tree as input (same pointer, not mutated)
}

// Runf returns a function that runs the testcase. visitor is the visitor to use
// in the test case, and initRoot optionally initializes the root of the tree before traversal.
func (tc *MutatingVisitorTestcase) Runf(
	visitor ast.Visitor,
	initRoot func(*ast.Root),
	options func() []cmp.Option) func(t *testing.T) {
	opts := []cmp.Option{resourcequota.ResourceQuantityEqual(), cmp.AllowUnexported(ast.Extension{}, discovery.APIInfo{})}
	if options != nil {
		opts = append(opts, options()...)
	}
	return func(t *testing.T) {
		copier := visitorpkg.NewCopying()
		copier.SetImpl(copier)
		if initRoot != nil {
			initRoot(tc.Input)
		}
		inputCopy := tc.Input.Accept(copier)

		actual := tc.Input.Accept(visitor)
		if !cmp.Equal(tc.Input, inputCopy, opts...) {
			t.Errorf("Input mutated while running visitor: %s", cmp.Diff(inputCopy, tc.Input, opts...))
		}

		err := visitor.Error()
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
		if tc.ExpectNoop {
			if tc.Input != actual {
				t.Fatalf("expected noop, mismatch on expected vs actual: %s", cmp.Diff(tc.ExpectOutput, actual, opts...))
			}
			tc.ExpectOutput = inputCopy
		}
		if !cmp.Equal(tc.ExpectOutput, actual, opts...) {
			t.Errorf("mismatch on expected vs actual:\ndiff:\n%s",
				cmp.Diff(tc.ExpectOutput, actual, opts...))
		}
	}
}

// MutatingVisitorTestcases specifies a list of testcases for the
type MutatingVisitorTestcases struct {
	// VisitorCtor returns a created visitor.
	VisitorCtor func() ast.Visitor
	// InitRoot initializes the root before tree traversal.  Skipped if nil.
	InitRoot func(r *ast.Root)
	// Options is the list of options to apply when comparing trees.  Using default if nil
	Options   func() []cmp.Option
	Testcases []MutatingVisitorTestcase
}

// Run runs all testcases.
func (tcs *MutatingVisitorTestcases) Run(t *testing.T) {
	t.Helper()
	for _, testcase := range tcs.Testcases {
		t.Run(testcase.Name, testcase.Runf(tcs.VisitorCtor(), tcs.InitRoot, tcs.Options))
	}
}
