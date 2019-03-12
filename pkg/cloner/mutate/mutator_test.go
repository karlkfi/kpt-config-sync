package mutate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/testing/object"
)

func TestObjects(t *testing.T) {
	const foo = "foo"
	const barGroup = "bar"
	const bar = barGroup + "/qux"

	testCases := []struct {
		name     string
		object   ast.FileObject
		mutators []Mutator
		expected ast.FileObject
	}{
		{
			name: "no mutations does nothing",
			object: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			expected: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
		},
		{
			name: "nil mutator does nothing",
			object: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []Mutator{nil},
			expected: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
		},
		{
			name: "remove annotation foo removes annotation",
			object: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []Mutator{RemoveAnnotation(foo)},
			expected: object.Build(kinds.Role(),
				object.Annotations(map[string]string{}),
				object.Label(bar, "true")),
		},
		{
			name: "remove bar label group removes label",
			object: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []Mutator{RemoveLabelGroup(barGroup)},
			expected: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Labels(map[string]string{})),
		},
		{
			name: "remove annotation foo and bar label group removes both",
			object: object.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []Mutator{RemoveAnnotation(foo), RemoveLabelGroup(barGroup)},
			expected: object.Build(kinds.Role(),
				object.Annotations(map[string]string{}),
				object.Labels(map[string]string{})),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := []ast.FileObject{tc.object}
			ApplyAll(actual, tc.mutators...)

			if diff := cmp.Diff(tc.expected, actual[0]); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
