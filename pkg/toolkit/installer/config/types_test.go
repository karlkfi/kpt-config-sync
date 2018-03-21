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
				"user": "someuser@example.com",
				"contexts": [
				  "foo",
				  "bar"
				]
			}`,
			expected: Config{
				User:     "someuser@example.com",
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: defaultSyncWaitTimeoutSeconds,
					SyncBranch:      "master",
				},
			},
		},
		{
			name: "Basic YAML",
			input: `user: someuser@example.com
contexts:
- foo
- bar
`,
			expected: Config{
				User:     "someuser@example.com",
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: defaultSyncWaitTimeoutSeconds,
					SyncBranch:      "master",
				},
			},
		},
		{
			name: "Example config",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"git": {
				"GIT_SYNC_REPO": "git@github.com:repo/example.git",
				"GIT_SYNC_BRANCH": "test",
				"GIT_SYNC_WAIT": 1,
				"ROOT_POLICY_DIR": "foo-corp"
		},
		"ssh": {
				"privateKeyFilename": "privateKey",
				"knownHostsFilename": "knownHosts"
		}
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
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
		{
			name: "$HOME substitution",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"git": {
				"GIT_SYNC_REPO": "git@github.com:repo/example.git",
				"GIT_SYNC_BRANCH": "test",
				"GIT_SYNC_WAIT": 1,
				"ROOT_POLICY_DIR": "foo-corp"
		},
		"ssh": {
				"privateKeyFilename": "$HOME/privateKey",
				"knownHostsFilename": "$HOME/knownHosts"
		}
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
					SyncBranch:      "test",
					RootPolicyDir:   "foo-corp",
					SyncRepo:        "git@github.com:repo/example.git",
				},
				Ssh: SshConfig{
					PrivateKeyFilename: "/home/user/privateKey",
					KnownHostsFilename: "/home/user/knownHosts",
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
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
				},
			},
		},
		{
			name: "Basic",
			input: Config{
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
				},
				Ssh: SshConfig{
					KnownHostsFilename: "/home/user/known_hosts",
					PrivateKeyFilename: "/home/user/private_key",
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
