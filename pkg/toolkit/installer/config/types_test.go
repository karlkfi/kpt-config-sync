package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRead(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Config
	}{
		{
			name: "Basic",
			input: `{
				"clusters": [
				  "foo",
				  "bar"
				]
			}`,
			expected: Config{
				Clusters: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: defaultSyncWaitTimeoutSeconds,
					SyncBranch:      "master",
				},
			},
		},
		{
			name: "Example config",
			input: `{
		"clusters": [
				"your_cluster"
		],
		"git": {
				"syncRepo": "git@github.com:repo/example.git",
				"syncBranch": "test",
				"syncWaitSeconds": 1,
				"rootPolicyDir": "foo-corp"
		},
		"ssh": {
				"privateKeyFilename": "privateKey",
				"knownHostsFilename": "knownHosts"
		}
			}`,
			expected: Config{
				Clusters: []string{"your_cluster"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
					SyncBranch:      "test",
					RootPolicyDir:   "foo-corp",
					SyncRepo:        "git@github.com:repo/example.git",
				},
				Ssh: SshConfig{
					PrivateKeyFilename: "privateKey",
					KnownHostsFilename: "knownHosts",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			c, err := Load(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !cmp.Equal(c, tt.expected) {
				t.Errorf("Load():\n%v\nwant: %v\ndiff: %v", c, tt.expected, cmp.Diff(tt.expected, c))
			}
		})
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected string
	}{
		{
			name: "Basic",
			input: Config{
				Clusters: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			err := tt.input.WriteInto(w)
			if err != nil {
				t.Fatalf("WriteInto(): unexpected error: %v", err)
			}
			c, err := Load(w)
			if err != nil {
				t.Fatalf("Load(): unexpected error: %v", err)
			}
			if !cmp.Equal(c, tt.input) {
				t.Errorf("Load():\n%v\nwant: %v\ndiff: %v", c, tt.input, cmp.Diff(tt.input, c))
			}
		})
	}
}
