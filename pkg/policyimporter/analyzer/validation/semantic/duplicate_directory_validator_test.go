package semantic

import (
	"path/filepath"
	"testing"

	"github.com/google/nomos/pkg/util/multierror"
)

type testCase struct {
	name    string
	dirs    [][]string
	isError bool
}

var testCases = []testCase{
	{
		name: "empty",
	},
	{
		name: "one dir",
		dirs: [][]string{{"a"}},
	},
	{
		name: "two different namespaces",
		dirs: [][]string{{"a"}, {"b"}},
	},
	{
		name: "parent and child",
		dirs: [][]string{{"a"}, {"a", "b"}},
	},
	{
		name:    "duplicates parent name",
		dirs:    [][]string{{"a", "a"}},
		isError: true,
	},
	{
		name:    "two same namespaces",
		dirs:    [][]string{{"a", "c"}, {"b", "c"}},
		isError: true,
	},
	{
		name:    "two same abstract namespaces",
		dirs:    [][]string{{"a", "c", "d"}, {"b", "c", "e"}},
		isError: true,
	},
	{
		name:    "same abstract namespace and namespace",
		dirs:    [][]string{{"a", "c", "d"}, {"b", "c"}},
		isError: true,
	},
}

func (tc testCase) Run(t *testing.T) {
	eb := multierror.Builder{}

	dirs := make([]string, len(tc.dirs))
	for i, dir := range tc.dirs {
		dirs[i] = filepath.Join(dir...)
	}

	DuplicateDirectoryValidator{dirs}.Validate(&eb)
	if eb.HasErrors() && !tc.isError {
		t.Fatalf("did not expect error %s", eb.Build())
	} else if !eb.HasErrors() && tc.isError {
		t.Fatalf("expected error")
	}
}

func TestDuplicateDirectoryValidator(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}
