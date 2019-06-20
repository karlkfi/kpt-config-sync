package parsertest

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/backend"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
)

// TestCase represents a test case that runs AST hydration on a set of already-parsed files.
// These test cases run steps such as vetting that configuration is valid and performing Abstract
// Namespace inheritance.
//
// If error codes are expected, list them in Errors.
// Otherwise, fill in the expected hydrated hierarchy in Expected.
type TestCase struct {
	// Name is the name of the test case.
	Name string

	// Objects is the array of objects to run hydration and validation on.
	Objects []ast.FileObject

	// Expected is the resulting expected Root. Ignored if Errors is non-nil.
	Expected *namespaceconfig.AllConfigs

	// Errors is the errors the test case expects, if any.
	Errors []string
}

// Test is a suite of tests to run on Parser's vetting and hydration functionality.
type Test struct {
	// NewParser returns a function which produces a Parser configured to hydrate a dry representation
	// of the hierarchy. This allows generating a new Parser for each test.
	NewParser func(t *testing.T) *filesystem.Parser

	// DefaultObjects is the list of objects implicitly included in every test case.
	DefaultObjects []ast.FileObject

	// TestCases is the list of test cases to run.
	TestCases []TestCase
}

// Failure represents a test case which is expected to fail with a single error code.
//
// TODO: Write method which allows specifying multiple errors, Failures().
// TODO: Write Success() function for when parsing succeeds.
func Failure(name string, err string, objects ...ast.FileObject) TestCase {
	return TestCase{
		Name:    name,
		Errors:  []string{err},
		Objects: objects,
	}
}

// NewParser generates a default Parser
func NewParser(t *testing.T) *filesystem.Parser {
	t.Helper()

	f := fstesting.NewTestClientGetter(t)
	defer func() {
		if err := f.Cleanup(); err != nil {
			t.Fatal(errors.Wrap(err, "could not clean up"))
		}
	}()

	return filesystem.NewParser(
		f,
		filesystem.ParserOpt{
			Vet:       true,
			Validate:  true,
			Extension: &filesystem.NomosVisitorProvider{},
		})
}

// RunAll runs each unit test.
func (pt Test) RunAll(t *testing.T) {
	t.Helper()

	for _, tc := range pt.TestCases {
		t.Run(tc.Name, func(t *testing.T) {
			parser := pt.NewParser(t)

			objects := append(pt.DefaultObjects, tc.Objects...)
			flatRoot := treetesting.BuildFlatTree(t, objects...)

			visitors := parser.GenerateVisitors(flatRoot, &namespaceconfig.AllConfigs{}, nil)
			outputVisitor := backend.NewOutputVisitor()
			visitors = append(visitors, outputVisitor)

			// TODO: Allow tests to use clusterName parameter.
			parser.HydrateRoot(visitors, "", time.Time{}, "")

			if tc.Errors != nil {
				vettesting.ExpectErrors(tc.Errors, parser.Errors(), t)
			} else {
				actual := outputVisitor.AllConfigs()
				if diff := cmp.Diff(tc.Expected, actual); diff != "" {
					t.Fatalf(diff)
				}
			}
		})
	}
}
