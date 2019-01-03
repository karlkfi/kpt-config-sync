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

type testCase string

func (tc testCase) Root() string {
	return filepath.Join("../../../", string(tc))
}

func TestParse(t *testing.T) {
	tests := []testCase{
		"examples/errors",
		"examples/parse-errors/empty-system-dir",
		"examples/parse-errors/illegal-namespace-sync",
		"examples/parse-errors/illegal-system-kind",
		"examples/parse-errors/invalid-crd-name",
		"examples/parse-errors/invalid-resources-sync",
		"examples/parse-errors/kind-with-multiple-versions",
		"examples/parse-errors/missing-repo",
		"examples/parse-errors/missing-system-dir",
		"examples/parse-errors/multiple-repos",
		"examples/parse-errors/unsupported-repo-version",
	}

	for _, test := range tests {
		t.Run(string(test), test.Run)
	}
}

func (tc *testCase) Run(t *testing.T) {
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
		t.Errorf("diff:\n%v", diff)
		t.Errorf(`If this change is correct, run:
make build
nomos vet --path=%[1]v --validate=false 2> %[1]v/expected-errs.txt`, *tc)
	}
}
