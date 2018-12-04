package filesystem

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/google/go-cmp/cmp"
	fstesting "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
)

type testCase struct {
	root string
}

func (tc testCase) Root() string {
	return filepath.Join("../../../", tc.root)
}

func TestParse(t *testing.T) {
	testCases := []testCase{
		{
			root: "examples/errors",
		},
		{
			root: "examples/parse-errors/empty-system-dir",
		},
		{
			root: "examples/parse-errors/illegal-namespace-sync",
		},
		{
			root: "examples/parse-errors/illegal-system-kind",
		},
		{
			root: "examples/parse-errors/invalid-crd-name",
		},
		{
			root: "examples/parse-errors/invalid-resources-sync",
		},
		{
			root: "examples/parse-errors/kind-with-multiple-versions",
		},
		{
			root: "examples/parse-errors/missing-repo",
		},
		{
			root: "examples/parse-errors/missing-system-dir",
		},
		{
			root: "examples/parse-errors/multiple-configmaps",
		},
		{
			root: "examples/parse-errors/multiple-repos",
		},
		{
			root: "examples/parse-errors/unsupported-repo-version",
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func runTestCase(t *testing.T, tc testCase) {
	var expected []string
	expectedFile, err := os.Open(filepath.Join(tc.Root(), "expected-errs.txt"))
	if err != nil && !os.IsNotExist(err) {
		t.Error(err)
		return
	}
	scanner := bufio.NewScanner(expectedFile)
	for scanner.Scan() {
		expectedLine := scanner.Text()
		expected = append(expected, expectedLine)
		if strings.Contains(expectedLine, " /") {
			t.Errorf("Test data MUST NOT have absolute paths:\n%s", expectedLine)
		}
	}

	f := fstesting.NewTestFactory()
	defer func() {
		if err := f.Cleanup(); err != nil {
			t.Fatal(errors.Wrap(err, "could not clean up"))
		}
	}()

	p, err2 := NewParserWithFactory(
		f,
		ParserOpt{
			Vet:      true,
			Validate: false,
		},
	)
	if err2 != nil {
		t.Fatalf("unexpected error: %#v", err2)
	}

	_, actualErrors := p.Parse(tc.Root())
	actual := strings.Split("Found issues: "+actualErrors.Error(), "\n")

	diff := cmp.Diff(expected, actual)

	if diff != "" {
		t.Errorf("Test Dir %q", tc.root)
		t.Error(diff)
		t.Errorf(`If this change is correct, run:
make build
nomos vet --path=%[1]s --validate=false 2> %[1]s/expected-errs.txt`, tc.root)
	}
}
