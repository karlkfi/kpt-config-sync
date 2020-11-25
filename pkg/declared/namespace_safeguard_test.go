package declared

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDontDeleteAllNamespaces(t *testing.T) {
	testCases := []struct {
		name     string
		previous []string
		current  []string
		want     status.Error
	}{
		{
			name:     "zero to zero",
			previous: []string{},
			current:  []string{},
		},
		{
			name:     "zero to one",
			previous: []string{},
			current:  []string{"foo"},
		},
		{
			name:     "zero to two",
			previous: []string{},
			current:  []string{"foo", "bar"},
		},
		{
			name:     "one to zero",
			previous: []string{},
			current:  []string{},
		},
		{
			name:     "one to one",
			previous: []string{"foo"},
			current:  []string{"foo"},
		},
		{
			name:     "one to two",
			previous: []string{"foo"},
			current:  []string{"foo", "bar"},
		},
		{
			name:     "two to zero",
			previous: []string{"foo", "bar"},
			current:  []string{},
			want:     DeleteAllNamespacesError([]string{"foo", "bar"}),
		},
		{
			name:     "two to one",
			previous: []string{"foo", "bar"},
			current:  []string{"foo"},
		},
		{
			name:     "two to two",
			previous: []string{"foo", "bar"},
			current:  []string{"foo", "bar"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			previous := make(map[core.ID]*unstructured.Unstructured)
			for _, p := range tc.previous {
				u := fake.UnstructuredObject(kinds.Namespace(), core.Name(p))
				previous[core.IDOf(u)] = u
			}
			current := make(map[core.ID]*unstructured.Unstructured)
			for _, c := range tc.current {
				u := fake.UnstructuredObject(kinds.Namespace(), core.Name(c))
				current[core.IDOf(u)] = u
			}

			got := deletesAllNamespaces(previous, current)
			if !errors.Is(got, tc.want) {
				t.Errorf("got deletesAllNamespaces() = %v, want %v", got, tc.want)
			}
		})
	}
}
