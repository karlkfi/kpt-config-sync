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
				Git: &GitConfig{
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
				Git: &GitConfig{
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
				Git: &GitConfig{
					SyncWaitSeconds: 1,
					SyncBranch:      "test",
					RootPolicyDir:   "foo-corp",
					SyncRepo:        "git@github.com:repo/example.git",
				},
				Ssh: &SshConfig{
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
				Git: &GitConfig{
					SyncWaitSeconds: 1,
					SyncBranch:      "test",
					RootPolicyDir:   "foo-corp",
					SyncRepo:        "git@github.com:repo/example.git",
				},
				Ssh: &SshConfig{
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
  GIT_SYNC_WAIT: 60
  ROOT_POLICY_DIR: foo-corp
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
				Git: &GitConfig{
					SyncWaitSeconds: 1,
				},
			},
			expected: `contexts:
- foo
- bar
git:
  GIT_SYNC_BRANCH: ""
  GIT_SYNC_REPO: ""
  GIT_SYNC_WAIT: 1
  ROOT_POLICY_DIR: ""
`,
		},
		{
			name: "Basic",
			input: Config{
				Contexts: []string{"foo", "bar"},
				Git: &GitConfig{
					SyncWaitSeconds: 1,
				},
				Ssh: &SshConfig{
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
  GIT_SYNC_WAIT: 1
  ROOT_POLICY_DIR: ""
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
			name: "no private key specified",
			config: Config{
				Ssh: &SshConfig{},
			},
			wantErr: errors.Errorf("ssh private key file name not specified"),
		},
		{
			name: "no git repo specified",
			config: Config{
				Git: &GitConfig{},
			},
			wantErr: errors.Errorf("git not repo specified"),
		},
		{
			name: "https uri w/ no keys specified",
			config: Config{
				Git: &GitConfig{
					SyncRepo: "https://foobar.com/foo-corp-example.git",
				},
			},
		},
		{
			name: "ssh uri w/ no keys specified",
			config: Config{
				Git: &GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
				},
			},
			wantErr: errors.Errorf("ssh path specified for git repo, but private key not specified"),
		},
		{
			name: "ssh uri w/ keys that don't exist specified",
			config: Config{
				Git: &GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
				},
				Ssh: &SshConfig{
					PrivateKeyFilename: "/some/fake/path/id_rsa",
				},
			},
			fileExists: testExists{false},
			wantErr:    errors.Errorf("ssh path specified for git repo, but private key doesn't exist: /some/fake/path/id_rsa"),
		},
		{
			name: "allow no funny characters in the file path beginning with /home/user",
			config: Config{
				Git: &GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
				},
				Ssh: &SshConfig{
					PrivateKeyFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{false},
			wantErr:    errors.Errorf("ssh path specified for git repo, but private key doesn't exist: /some/fake/path/id_rsa"),
		},
		{
			name: "ssh uri w/ keys that exist specified",
			config: Config{
				Git: &GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
				},
				Ssh: &SshConfig{
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
