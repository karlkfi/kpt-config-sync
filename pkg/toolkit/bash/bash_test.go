package bash

import (
	"context"
	"strings"
	"testing"

	"github.com/google/stolos/pkg/toolkit/exec"
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
