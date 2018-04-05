package bash

import (
	"context"
	"strings"
	"testing"

	"github.com/google/nomos/pkg/process/exec"
)

func TestRunWithEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{
			name: "Basic",
			env:  "KEY=VALUE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec.SetFakeOutputsForTest(strings.NewReader(""), nil, nil)
			env, err := runWithEnv(context.Background(), "dummy", tt.env)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if env[0] != tt.env {
				t.Errorf("env: %v, wanted: %v", env, tt.env)
			}
		})
	}
}

func TestRunWithEnvError(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{
			name: "Basic",
			env:  "KEY=VALUE",
		},
	}
	// Ensure that a real bash command is used for this test.
	exec.SetFakeOutputsForTest(nil, nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runWithEnv(context.Background(), "SOME_NONEXISTENT_SCRIPT", tt.env)
			if err == nil {
				t.Errorf("expected error")
			}
		})
	}
}
