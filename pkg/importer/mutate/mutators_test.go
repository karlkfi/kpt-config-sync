package mutate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/testing/object"
)

func TestRemoveAnnotation(t *testing.T) {
	testCases := []struct {
		name       string
		object     ast.FileObject
		annotation string
		expected   ast.FileObject
	}{
		{
			name:       "foo removes annotation foo",
			object:     object.Build(kinds.Role(), object.Annotation("foo", "")),
			annotation: "foo",
			expected:   object.Build(kinds.Role()),
		},
		{
			name:       "foo does not remove annotation bar",
			object:     object.Build(kinds.Role(), object.Annotation("bar", "")),
			annotation: "foo",
			expected:   object.Build(kinds.Role(), object.Annotation("bar", "")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := []ast.FileObject{tc.object}

			RemoveAnnotation(tc.annotation).Apply(actual)

			if diff := cmp.Diff(tc.expected, actual[0]); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestRemoveAnnotationGroup(t *testing.T) {
	testCases := []struct {
		name     string
		object   ast.FileObject
		group    string
		expected ast.FileObject
	}{
		{
			name:     "foo removes annotation foo/bar",
			object:   object.Build(kinds.Role(), object.Annotation("foo/bar", "")),
			group:    "foo",
			expected: object.Build(kinds.Role()),
		},
		{
			name:     "foo does not remove annotation bar/foo",
			object:   object.Build(kinds.Role(), object.Annotation("bar/foo", "")),
			group:    "foo",
			expected: object.Build(kinds.Role(), object.Annotation("bar/foo", "")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := []ast.FileObject{tc.object}

			RemoveAnnotationGroup(tc.group).Apply(actual)

			if diff := cmp.Diff(tc.expected, actual[0]); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestRemoveLabel(t *testing.T) {
	testCases := []struct {
		name     string
		object   ast.FileObject
		label    string
		expected ast.FileObject
	}{
		{
			name:     "foo removes label foo",
			object:   object.Build(kinds.Role(), object.Label("foo", "")),
			label:    "foo",
			expected: object.Build(kinds.Role()),
		},
		{
			name:     "foo does not remove label bar",
			object:   object.Build(kinds.Role(), object.Label("bar", "")),
			label:    "foo",
			expected: object.Build(kinds.Role(), object.Label("bar", "")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := []ast.FileObject{tc.object}

			RemoveLabel(tc.label).Apply(actual)

			if diff := cmp.Diff(tc.expected, actual[0]); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestRemoveLabelGroup(t *testing.T) {
	testCases := []struct {
		name     string
		object   ast.FileObject
		group    string
		expected ast.FileObject
	}{
		{
			name:     "foo removes label foo/bar",
			object:   object.Build(kinds.Role(), object.Label("foo/bar", "")),
			group:    "foo",
			expected: object.Build(kinds.Role()),
		},
		{
			name:     "foo does not remove label bar/foo",
			object:   object.Build(kinds.Role(), object.Label("bar/foo", "")),
			group:    "foo",
			expected: object.Build(kinds.Role(), object.Label("bar/foo", "")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := []ast.FileObject{tc.object}

			RemoveLabelGroup(tc.group).Apply(actual)

			if diff := cmp.Diff(tc.expected, actual[0]); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
