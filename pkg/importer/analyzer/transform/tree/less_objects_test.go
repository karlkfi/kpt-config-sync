package tree

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestLessObjects(t *testing.T) {
	testCases := []struct {
		name             string
		i                ast.FileObject
		j                ast.FileObject
		expectedLessThan bool
	}{
		{
			name:             "i Group less than j",
			i:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Group: "A"}, ""),
			j:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Group: "B"}, ""),
			expectedLessThan: true,
		},
		{
			name:             "j Group less than i",
			i:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Group: "B"}, ""),
			j:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Group: "A"}, ""),
			expectedLessThan: false,
		},
		{
			name:             "i Version less than j",
			i:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Version: "1"}, ""),
			j:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Version: "2"}, ""),
			expectedLessThan: true,
		},
		{
			name:             "j Version less than i",
			i:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Version: "2"}, ""),
			j:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Version: "1"}, ""),
			expectedLessThan: false,
		},
		{
			name:             "i Kind less than j",
			i:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Kind: "A"}, ""),
			j:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Kind: "B"}, ""),
			expectedLessThan: true,
		},
		{
			name:             "j Kind less than i",
			i:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Kind: "B"}, ""),
			j:                asttesting.NewFakeFileObject(schema.GroupVersionKind{Kind: "A"}, ""),
			expectedLessThan: false,
		},
		{
			name:             "i Name less than j",
			i:                *ast.ParseFileObject(asttesting.NewFakeObject(schema.GroupVersionKind{}).WithName("A")),
			j:                *ast.ParseFileObject(asttesting.NewFakeObject(schema.GroupVersionKind{}).WithName("B")),
			expectedLessThan: true,
		},
		{
			name:             "j Name less than i",
			i:                *ast.ParseFileObject(asttesting.NewFakeObject(schema.GroupVersionKind{}).WithName("B")),
			j:                *ast.ParseFileObject(asttesting.NewFakeObject(schema.GroupVersionKind{}).WithName("A")),
			expectedLessThan: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.MetaObject().GetUID()

			actualLessThan := lessFileObject(tc.i, tc.j)

			if tc.expectedLessThan {
				if !actualLessThan {
					t.Fatal("expected 'i' element to be less than 'j' element")
				}
			} else {
				if actualLessThan {
					t.Fatal("expected 'j' element to be less than 'i' element")
				}
			}
		})
	}
}
