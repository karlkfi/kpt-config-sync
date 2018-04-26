package config

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
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
					UseSSH:          true,
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
					UseSSH:          true,
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
				"GIT_SYNC_SSH": true,
				"GIT_SYNC_BRANCH": "test",
				"GIT_SYNC_WAIT": 1,
				"POLICY_DIR": "foo-corp"
		},
		"ssh": {
				"privateKeyFilename": "privateKey",
				"knownHostsFilename": "knownHosts"
		}
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				Git: GitConfig{
					UseSSH:          true,
					SyncWaitSeconds: 1,
					SyncBranch:      "test",
					RootPolicyDir:   "foo-corp",
					SyncRepo:        "git@github.com:repo/example.git",
				},
				SSH: SSHConfig{
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
				"POLICY_DIR": "foo-corp"
		},
		"ssh": {
				"privateKeyFilename": "$HOME/privateKey",
				"knownHostsFilename": "$HOME/knownHosts"
		}
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				Git: GitConfig{
					UseSSH:          true,
					SyncWaitSeconds: 1,
					SyncBranch:      "test",
					RootPolicyDir:   "foo-corp",
					SyncRepo:        "git@github.com:repo/example.git",
				},
				SSH: SSHConfig{
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

func TestUnprintable(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "With explicit unprintable characters",
			input: "ssh:\n \xc2knownHostsFilename: /somefile",
		},
		{
			// Don't be deceived: the text below contains unprintable characters,
			// for example before the key knownHostsFilename.
			name: "With unprintable characters",
			input: `user: someuser@example.com
contexts:
- cluster-2
git:
  GIT_SYNC_BRANCH: master
  GIT_SYNC_REPO: git@github.com:frankfarzan/foo-corp-example.git
  GIT_SYNC_SSH: true
  GIT_SYNC_WAIT: 60
  POLICY_DIR: foo-corp
ssh:
  knownHostsFilename: $HOME/.ssh/known_hosts
  privateKeyFilename: $HOME/.ssh/id_rsa.nomos
user: someuser@google.com
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			_, err := Load(r)
			if err == nil {
				t.Errorf("expected YAML decoding error")
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
				SSH: SSHConfig{},
			},
			expected: `contexts:
- foo
- bar
git:
  GIT_SYNC_BRANCH: ""
  GIT_SYNC_REPO: ""
  GIT_SYNC_SSH: false
  GIT_SYNC_WAIT: 1
  POLICY_DIR: ""
ssh: {}
`,
		},
		{
			name: "Basic",
			input: Config{
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
					UseSSH:          true,
				},
				SSH: SSHConfig{
					KnownHostsFilename: "/home/user/known_hosts",
					PrivateKeyFilename: "/home/user/private_key",
				},
			},
			expected: `contexts:
- foo
- bar
git:
  GIT_SYNC_BRANCH: ""
  GIT_SYNC_REPO: ""
  GIT_SYNC_SSH: true
  GIT_SYNC_WAIT: 1
  POLICY_DIR: ""
ssh:
  knownHostsFilename: $HOME/known_hosts
  privateKeyFilename: $HOME/private_key
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := bytes.NewBuffer(nil)
			err := tt.input.WriteInto(w)
			if err != nil {
				t.Fatalf("WriteInto(): unexpected error: %v", err)
			}
			if got := w.String(); !cmp.Equal(got, tt.expected) {
				t.Errorf("WriteInto():\n%v\nwant: %v\ndiff: %v", got, tt.expected, cmp.Diff(got, tt.expected))
			}
		})
	}
}

func TestImmutable(t *testing.T) {
	tests := []struct {
		name          string
		cfg, expected Config
	}{
		{
			name: "SSHConfig",
			cfg: Config{
				SSH: SSHConfig{"/home/user/file1", "/home/user/file2"},
			},
			expected: Config{
				SSH: SSHConfig{"/home/user/file1", "/home/user/file2"},
			},
		},
		{
			name: "GitConfig",
			cfg: Config{
				Git: GitConfig{
					SyncRepo:        "some_repo",
					SyncBranch:      "some_branch",
					RootPolicyDir:   "some_root_policy_dir",
					SyncWaitSeconds: 100,
				},
			},
			expected: Config{
				Git: GitConfig{
					SyncRepo:        "some_repo",
					SyncBranch:      "some_branch",
					RootPolicyDir:   "some_root_policy_dir",
					SyncWaitSeconds: 100,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.WriteInto(bytes.NewBuffer(nil))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !cmp.Equal(tt.cfg, tt.expected) {
				t.Errorf("WriteInto() changed source; got %v, want: %v, diff: %v", tt.cfg, tt.expected, cmp.Diff(tt.cfg, tt.expected))
			}

		})

	}
}

type testExists struct {
	exists bool
}

// Check implements FileExists.
func (s testExists) Check(name string) bool {
	if strings.Contains(name, "$") {
		panic(fmt.Sprintf("Check: file has unexpected characters: %q", name))
	}
	return s.exists
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		fileExists FileExists
		config     Config
		wantErr    error
	}{
		{
			name: "no git repo specified",
			config: Config{
				Git: GitConfig{
					SyncBranch: "master",
				},
			},
			wantErr: errors.Errorf("git not repo specified"),
		},
		{
			name: "https uri w/ no keys specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "https://foobar.com/foo-corp-example.git",
					UseSSH:   false,
				},
			},
		},
		{
			name: "ssh uri w/ no keys specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
			},
			wantErr: errors.Errorf("ssh path specified for git repo, but private key not specified"),
		},
		{
			name: "ssh uri w/ keys that don't exist specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
				SSH: SSHConfig{
					PrivateKeyFilename: "/some/fake/path/id_rsa",
				},
			},
			fileExists: testExists{false},
			wantErr:    errors.Errorf("ssh path specified for git repo, but private key doesn't exist: /some/fake/path/id_rsa"),
		},
		{
			name: "allow no funny characters in the file path beginning with /home/user",
			config: Config{
				Git: GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
				SSH: SSHConfig{
					PrivateKeyFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{false},
			wantErr:    errors.Errorf("ssh path specified for git repo, but private key doesn't exist: /some/fake/path/id_rsa"),
		},
		{
			name: "non-ssh uri with UseSSH specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "https://foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
				SSH: SSHConfig{
					PrivateKeyFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{true},
			wantErr:    errors.Errorf("ssh not specified for ssh git repo url"),
		},
		{
			name: "ssh uri/UseSSH specified, no private key specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
				SSH: SSHConfig{
					KnownHostsFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{true},
			wantErr:    errors.Errorf("ssh not specified for ssh git repo url"),
		},
		{
			name: "ssh uri w/ keys that exist specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
				SSH: SSHConfig{
					PrivateKeyFilename: "/some/valid/path/id_rsa",
				},
			},
			fileExists: testExists{true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.fileExists)
			if (err != nil && tt.wantErr == nil) || (err == nil && tt.wantErr != nil) {
				t.Fatalf("Unexpected error when validating:\n%v\nwant: %v", err, tt.wantErr)
			}
		})
	}
}
