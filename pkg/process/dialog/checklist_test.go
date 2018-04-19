package dialog

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/process/exec"
)

func TestChecklist(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "Empty",
			output:   "",
			expected: []string{},
		},
		{
			name:     "Even emptier",
			output:   "  \t   \n  \t\n \t\t\n\t",
			expected: []string{},
		},
		{
			name:     "foo and bar selected",
			output:   "foo bar",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "wonky output is sanitized",
			output:   "  \tfoo\t\nbar\n\n   ",
			expected: []string{"foo", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Due to the way exec works, this can't be ran in parallel.
			exec.SetFakeOutputsForTest(nil, strings.NewReader(tt.output), nil)
			c := NewChecklist()
			c.Display()
			sel, err := c.Close()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !cmp.Equal(sel, tt.expected) {
				t.Errorf("Close()=%+v, want %+v, diff:\n%v", sel, tt.expected, cmp.Diff(tt.expected, sel))
			}
		})
	}
}
