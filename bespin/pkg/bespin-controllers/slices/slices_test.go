package slices

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestContainsString(t *testing.T) {
	testCases := []struct {
		want    bool
		strings []string
	}{
		{
			false, []string{},
		},
		{
			true, []string{"foo", "", "bar", " "},
		},
		{
			false, []string{"FOO", "", "BAR", " ", " bar", "bar ", "b ar"},
		},
	}
	mystr := "bar"
	for _, c := range testCases {
		got := ContainsString(c.strings, mystr)
		if got != c.want {
			t.Errorf("error in ContainsString function with slice: %v, and test string: %s. Got: %v, Want: %v.",
				c.strings, mystr, got, c.want)
		}
	}
}

func TestRemoveString(t *testing.T) {
	testCases := []struct {
		wantedSlice   []string
		originalSlice []string
	}{
		{
			[]string{}, []string{},
		},
		{
			[]string{"foo", "", " ", " bar", "bar ", "b ar", "FOO", "BAR"},
			[]string{"foo", "bar", "", " ", " bar", "bar ", "b ar", "FOO", "BAR"},
		},
		{
			[]string{"foo"}, []string{"bar", "foo", "bar"},
		},
	}
	mystr := "bar"
	for _, c := range testCases {
		gotSlice := RemoveString(c.originalSlice, mystr)
		sort.Strings(gotSlice)
		sort.Strings(c.wantedSlice)
		if !cmp.Equal(gotSlice, c.wantedSlice) {
			t.Errorf("error in RemoveString function with slice: %v, and test string: %s. Got: %v, Want: %v.",
				c.originalSlice, mystr, gotSlice, c.wantedSlice)
		}
	}
}
