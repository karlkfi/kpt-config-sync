package dialog

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/process/exec"
)

func TestMenu(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "Empty",
			output:   "",
			expected: "",
		},
		{
			name:     "foo selected",
			output:   "foo",
			expected: "foo",
		},
		{
			name:     "wonky foo selected",
			output:   " foo ",
			expected: " foo ",
		},
		{
			name:     "wonky output is not sanitized, sorry",
			output:   "  \tfoo\t\n\n   ",
			expected: "  \tfoo\t\n\n   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Due to the way exec works, this can't be ran in parallel.
			exec.SetFakeOutputsForTest(nil, strings.NewReader(tt.output), nil)
			m := NewMenu()
			m.Display()
			sel, err := m.Close()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if sel != tt.expected {
				t.Errorf("Close()=%+v, want %+v, diff:\n%v", sel, tt.expected, cmp.Diff(tt.expected, sel))
			}
		})
	}
}
