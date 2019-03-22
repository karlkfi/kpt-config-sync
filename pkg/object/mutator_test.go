package object_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/mutate"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestObjects(t *testing.T) {
	const foo = "foo"
	const barGroup = "bar"
	const bar = barGroup + "/qux"

	testCases := []struct {
		name     string
		object   ast.FileObject
		mutators []object.Mutator
		expected ast.FileObject
	}{
		{
			name: "no mutations does nothing",
			object: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			expected: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
		},
		{
			name: "nil mutator does nothing",
			object: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []object.Mutator{nil},
			expected: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
		},
		{
			name: "remove annotation foo removes annotation",
			object: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []object.Mutator{mutate.RemoveAnnotation(foo)},
			expected: fake.Build(kinds.Role(),
				object.Annotations(map[string]string{}),
				object.Label(bar, "true")),
		},
		{
			name: "remove bar label group removes label",
			object: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []object.Mutator{mutate.RemoveLabelGroup(barGroup)},
			expected: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Labels(map[string]string{})),
		},
		{
			name: "remove annotation foo and bar label group removes both",
			object: fake.Build(kinds.Role(),
				object.Annotation(foo, "true"),
				object.Label(bar, "true")),
			mutators: []object.Mutator{mutate.RemoveAnnotation(foo), mutate.RemoveLabelGroup(barGroup)},
			expected: fake.Build(kinds.Role(),
				object.Annotations(map[string]string{}),
				object.Labels(map[string]string{})),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := []ast.FileObject{tc.object}
			object.Mutate(tc.mutators...).Apply(actual)

			if diff := cmp.Diff(tc.expected, actual[0]); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
