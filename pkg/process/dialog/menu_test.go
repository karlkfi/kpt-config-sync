package dialog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/process/exec"
)

func TestMenu(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		outerr   string
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
		{
			name:     "user pressing cancel",
			output:   "",
			outerr:   "exit status 1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Due to the way exec works, this can't be ran in parallel.
			var fakeerr error
			if tt.outerr != "" {
				fakeerr = fmt.Errorf(tt.outerr)
			}
			exec.SetFakeOutputsForTest(nil, strings.NewReader(tt.output), fakeerr)
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
