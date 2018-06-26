package main

import (
	"testing"
)

// nolint:deadcode
func TestVersionOrDie(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "0.0.0",
			expected: "0.0.0",
		},
		{
			input:    "1.0.0",
			expected: "1.0.0",
		},
		{
			input:    "1.0.0-a-b-c-d-e-f-g",
			expected: "1.0.0-a-b-c-d-e-f-g",
		},
		{
			input:    "someinitialjunk1.0.0-a-b-c-d-e-f-g",
			expected: "1.0.0-a-b-c-d-e-f-g",
		},
		{
			input:    "v0.2.0-41-ge046a5f-dirty",
			expected: "0.2.0-41-ge046a5f-dirty",
		},
		{
			input:    "v0.2.0-41-ge046a5f-dirty-0123456",
			expected: "0.2.0-41-ge046a5f-dirty-0123456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v := versionOrDie(tt.input)
			if v.String() != tt.expected {
				t.Errorf("was: %v, want: %v", v, tt.expected)
			}
		})
	}
}
