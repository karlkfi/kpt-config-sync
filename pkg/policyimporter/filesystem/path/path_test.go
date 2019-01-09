package path

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type nomosRootTestCase struct {
	name string
	root string
}

var nomosRootTestCases = []nomosRootTestCase{
	{
		name: "unchanged",
		root: "/foo/bar",
	},
	{
		name: "relative becomes absolute",
		root: "foo/bar",
	},
	{
		name: "unclean absolute becomes clean",
		root: "//foo//bar",
	},
	{
		name: "unclean relative becomes clean",
		root: "foo//bar",
	},
}

func (tc nomosRootTestCase) Run(t *testing.T) {
	r, err := NewNomosRoot(tc.root)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(r.path) {
		t.Fatalf("result path is not absolute: %s", r.path)
	}
	if filepath.Clean(r.path) != r.path {
		t.Fatalf("result path is not clean: %s", r.path)
	}
}

func TestNewNomosRootPath(t *testing.T) {
	for _, tc := range nomosRootTestCases {
		t.Run(tc.name, tc.Run)
	}
}

func toRoot(root string, t *testing.T) NomosRoot {
	r, err := NewNomosRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type nomosRelativeTestCase struct {
	name     string
	relative string
	expected string
}

var nomosRelativeTestCases = []nomosRelativeTestCase{
	{
		name:     "unchanged",
		relative: "foo/bar",
		expected: "foo/bar",
	},
	{
		name:     "cleaned",
		relative: "foo//bar",
		expected: "foo/bar",
	},
}

func (tc nomosRelativeTestCase) Run(t *testing.T) {
	actual := toRoot(".", t).Join(tc.relative)
	expected := toRoot(".", t).Join(tc.expected)

	if diff := cmp.Diff(actual, expected, CmpNomosRelativePath()); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(actual.RelativeSlashPath(), tc.expected); diff != "" {
		t.Fatal(diff)
	}
}

func TestNomosRootPath_Join(t *testing.T) {
	for _, tc := range nomosRelativeTestCases {
		t.Run(tc.name, tc.Run)
	}
}

type cmpNomosRelativePathTestCase struct {
	name  string
	root1 string
	path1 string
	root2 string
	path2 string
	equal bool
}

var cmpNomosRelativePathTestCases = []cmpNomosRelativePathTestCase{
	{
		name:  "identical",
		root1: "root",
		path1: "path",
		root2: "root",
		path2: "path",
		equal: true,
	},
	{
		name:  "root different",
		root1: "root",
		path1: "path",
		root2: "root2",
		path2: "path",
		equal: false,
	},
	{
		name:  "path different",
		root1: "root",
		path1: "path",
		root2: "root",
		path2: "path2",
		equal: false,
	},
	{
		name:  "root and path different",
		root1: "root",
		path1: "path",
		root2: "root2",
		path2: "path2",
		equal: false,
	},
	{
		name:  "root and path different but full path same",
		root1: "root/foo",
		path1: "path",
		root2: "root",
		path2: "foo/path",
		equal: false,
	},
}

func (tc cmpNomosRelativePathTestCase) Run(t *testing.T) {
	path1 := toRoot(tc.root1, t).Join(tc.path1)
	path2 := toRoot(tc.root2, t).Join(tc.path2)

	if tc.equal {
		if diff := cmp.Diff(path1, path2, CmpNomosRelativePath()); diff != "" {
			t.Fatal(diff)
		}
	} else {
		if cmp.Equal(path1, path2, CmpNomosRelativePath()) {
			t.Fatal("paths unexpectedly equal")
		}
	}
}

func TestCmpNomosRelativePath(t *testing.T) {
	for _, tc := range cmpNomosRelativePathTestCases {
		t.Run(tc.name, tc.Run)
	}
}

func TestNewFakeNomosRelativePath(t *testing.T) {
	foo := "foo"
	fake := NewFakeNomosRelativePath(foo)

	if diff := cmp.Diff(foo, fake.RelativeSlashPath()); diff != "" {
		t.Fatal(diff)
	}
}

// CmpNomosRelativePath returns a cmp.Option for NomosRoot.
func CmpNomosRelativePath() cmp.Option {
	return cmp.Comparer(func(lhs, rhs NomosRelative) bool {
		return lhs.path == rhs.path && cmp.Equal(lhs.root, rhs.root, CmpNomosRootPath())
	})
}

// CmpNomosRootPath returns a cmp.Option for NomosRoot.
func CmpNomosRootPath() cmp.Option {
	return cmp.Comparer(func(lhs, rhs NomosRoot) bool {
		return lhs.path == rhs.path
	})
}

type relTestCase struct {
	name     string
	root     string
	targpath string
	expected string
}

var relTestCases = []relTestCase{
	{
		name:     "standard",
		root:     "foo",
		targpath: "foo/bar/qux",
		expected: "bar/qux",
	},
	{
		name:     "cleaned",
		root:     "foo",
		targpath: "foo//bar///qux",
		expected: "bar/qux",
	},
}

func (tc relTestCase) Run(t *testing.T) {
	root := toRoot(tc.root, t)
	abs, err := filepath.Abs(tc.targpath)
	if err != nil {
		t.Fatal(err)
	}
	path, err := root.Rel(abs)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(tc.expected, path.RelativeSlashPath()); diff != "" {
		t.Fatal(diff)
	}
}

func TestNomosRootPath_Rel(t *testing.T) {
	for _, tc := range relTestCases {
		t.Run(tc.name, tc.Run)
	}
}
