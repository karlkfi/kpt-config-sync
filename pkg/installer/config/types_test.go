package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRead(t *testing.T) {
	os.Setenv("HOME", "/home/user")
	tests := []struct {
		name     string
		input    string
		expected Config
	}{
		{
			name: "git: basic JSON",
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
			name: "git: basic YAML",
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
			name: "git: example config",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"git": {
				"GIT_SYNC_REPO": "git@github.com:repo/example.git",
				"GIT_SYNC_SSH": true,
				"PRIVATE_KEY_FILENAME": "privateKey",
				"KNOWN_HOSTS_FILENAME": "knownHosts",
				"GIT_SYNC_BRANCH": "test",
				"GIT_SYNC_WAIT": 1,
				"POLICY_DIR": "foo-corp",
				"GIT_COOKIE_FILENAME": "gitcookies"
		},
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				Git: GitConfig{
					UseSSH:             true,
					PrivateKeyFilename: "privateKey",
					KnownHostsFilename: "knownHosts",
					SyncWaitSeconds:    1,
					SyncBranch:         "test",
					RootPolicyDir:      "foo-corp",
					SyncRepo:           "git@github.com:repo/example.git",
					CookieFilename:     "gitcookies",
				},
			},
		},
		{
			name: "git: $HOME substitution",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"git": {
				"GIT_SYNC_REPO": "git@github.com:repo/example.git",
				"PRIVATE_KEY_FILENAME": "$HOME/privateKey",
				"KNOWN_HOSTS_FILENAME": "$HOME/knownHosts",
				"GIT_COOKIE_FILENAME": "$HOME/cookieFilename",
				"GIT_SYNC_BRANCH": "test",
				"GIT_SYNC_WAIT": 1,
				"POLICY_DIR": "foo-corp"
		},
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				Git: GitConfig{
					UseSSH:             true,
					PrivateKeyFilename: "/home/user/privateKey",
					KnownHostsFilename: "/home/user/knownHosts",
					CookieFilename:     "/home/user/cookieFilename",
					SyncWaitSeconds:    1,
					SyncBranch:         "test",
					RootPolicyDir:      "foo-corp",
					SyncRepo:           "git@github.com:repo/example.git",
				},
			},
		},
		{
			name: "gcp",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"gcp": {
				"ORG_ID": "1234",
				"PRIVATE_KEY_FILENAME": "$HOME/privateKey",
		},
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				GCP: GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/home/user/privateKey",
				},
				Git: GitConfig{
					UseSSH:          true,
					SyncWaitSeconds: 15,
					SyncBranch:      "master",
				},
			},
		},
		{
			name: "gcp: policy API address",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"gcp": {
				"ORG_ID": "1234",
				"PRIVATE_KEY_FILENAME": "$HOME/privateKey",
				"POLICY_API_ADDRESS": "localhost:1234",
		},
			}`,
			expected: Config{
				Contexts: []string{"your_cluster"},
				GCP: GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/home/user/privateKey",
					PolicyAPIAddress:   "localhost:1234",
				},
				Git: GitConfig{
					UseSSH:          true,
					SyncWaitSeconds: 15,
					SyncBranch:      "master",
				},
			},
		},
		{
			name: "cluster_id",
			input: `{
		"contexts": [
				"your_cluster"
		],
		"gcp": {
				"ORG_ID": "1234",
				"PRIVATE_KEY_FILENAME": "$HOME/privateKey",
		},
		"clusters": [
		  {
				"name": "other_cluster_1",
				"context": "other_cluster_context_1"
	      },
		  {
				"name": "other_cluster_2",
				"context": "other_cluster_context_2"
	      }
		]
			}`,
			expected: Config{
				Contexts: []string{"other_cluster_context_1", "other_cluster_context_2"},
				GCP: GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/home/user/privateKey",
				},
				Git: GitConfig{
					UseSSH:          true,
					SyncWaitSeconds: 15,
					SyncBranch:      "master",
				},
				Clusters: []Cluster{
					Cluster{Name: "other_cluster_1", Context: "other_cluster_context_1"},
					Cluster{Name: "other_cluster_2", Context: "other_cluster_context_2"},
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
			c = c.ExpandVarsCopy()
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
			input: "git:\n \xc2KNOWN_HOSTS_FILENAME: /somefile",
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
  KNOWN_HOSTS_FILENAME: $HOME/.ssh/known_hosts
  PRIVATE_KEY_FILENAME: $HOME/.ssh/id_rsa.nomos
  GIT_SYNC_WAIT: 60
  POLICY_DIR: foo-corp

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
			name: "Git",
			input: Config{
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds: 1,
				},
			},
			expected: `contexts:
- foo
- bar
gcp:
  ORG_ID: ""
  PRIVATE_KEY_FILENAME: ""
git:
  GIT_SYNC_BRANCH: ""
  GIT_SYNC_REPO: ""
  GIT_SYNC_SSH: false
  GIT_SYNC_WAIT: 1
  POLICY_DIR: ""
`,
		},
		{
			name: "Git with file paths",
			input: Config{
				Contexts: []string{"foo", "bar"},
				Git: GitConfig{
					SyncWaitSeconds:    1,
					UseSSH:             true,
					KnownHostsFilename: "/home/user/known_hosts",
					PrivateKeyFilename: "/home/user/private_key",
				},
			},
			expected: `contexts:
- foo
- bar
gcp:
  ORG_ID: ""
  PRIVATE_KEY_FILENAME: ""
git:
  GIT_SYNC_BRANCH: ""
  GIT_SYNC_REPO: ""
  GIT_SYNC_SSH: true
  GIT_SYNC_WAIT: 1
  KNOWN_HOSTS_FILENAME: /home/user/known_hosts
  POLICY_DIR: ""
  PRIVATE_KEY_FILENAME: /home/user/private_key
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
			name: "GitConfig ssh settings",
			cfg: Config{
				Git: GitConfig{
					PrivateKeyFilename: "/home/user/file1",
					KnownHostsFilename: "/home/user/file2",
				},
			},
			expected: Config{
				Git: GitConfig{
					PrivateKeyFilename: "/home/user/file1",
					KnownHostsFilename: "/home/user/file2",
				},
			},
		},
		{
			name: "GitConfig other settings",
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
		wantErr    bool
	}{
		{
			name:    "Neither git or gcp specified",
			config:  Config{},
			wantErr: true,
		},
		{
			name: "Both git or gcp specified",
			config: Config{
				Git: GitConfig{
					SyncRepo:           "git@foobar.com/foo-corp-example.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/valid/path/id_rsa",
				},
				GCP: GCPConfig{
					PrivateKeyFilename: "/some/valid/path/id_rsa",
					OrgID:              "123",
				},
			},
			fileExists: testExists{true},
			wantErr:    true,
		},
		{
			name: "git: no git repo specified",
			config: Config{
				Git: GitConfig{
					SyncBranch:    "master",
					RootPolicyDir: "foo",
				},
			},
			wantErr: true,
		},
		{
			name: "git: https uri w/ no keys specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "https://foobar.com/foo-corp-example.git",
					UseSSH:   false,
				},
			},
		},
		{
			name: "git: ssh uri w/ no keys specified",
			config: Config{
				Git: GitConfig{
					SyncRepo: "git@foobar.com/foo-corp-example.git",
					UseSSH:   true,
				},
			},
			wantErr: true,
		},
		{
			name: "git: ssh uri w/ keys that don't exist specified",
			config: Config{
				Git: GitConfig{
					SyncRepo:           "git@foobar.com/foo-corp-example.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/fake/path/id_rsa",
				},
			},
			fileExists: testExists{false},
			wantErr:    true,
		},
		{
			name: "git: allow no funny characters in the file path beginning with /home/user",
			config: Config{
				Git: GitConfig{
					SyncRepo:           "git@foobar.com/foo-corp-example.git",
					UseSSH:             true,
					PrivateKeyFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{false},
			wantErr:    true,
		},
		{
			name: "git: non-ssh uri with UseSSH specified",
			config: Config{
				Git: GitConfig{
					SyncRepo:           "https://foobar.com/foo-corp-example.git",
					UseSSH:             true,
					PrivateKeyFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{true},
			wantErr:    true,
		},
		{
			name: "git: ssh uri/UseSSH specified, no private key specified",
			config: Config{
				Git: GitConfig{
					SyncRepo:           "git@foobar.com/foo-corp-example.git",
					UseSSH:             true,
					KnownHostsFilename: "/home/user/path/id_rsa",
				},
			},
			fileExists: testExists{true},
			wantErr:    true,
		},
		{
			name: "git: ssh uri w/ keys that exist specified",
			config: Config{
				Git: GitConfig{
					SyncRepo:           "git@foobar.com/foo-corp-example.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/valid/path/id_rsa",
				},
			},
			fileExists: testExists{true},
		},
		{
			name: "gcp: valid",
			config: Config{
				GCP: GCPConfig{
					PrivateKeyFilename: "/some/valid/path/id_rsa",
					OrgID:              "123",
				},
			},
			fileExists: testExists{true},
		},
		{
			name: "gcp: with policy API address",
			config: Config{
				GCP: GCPConfig{
					PrivateKeyFilename: "/some/valid/path/id_rsa",
					OrgID:              "123",
					PolicyAPIAddress:   "localhost:1234",
				},
			},
			fileExists: testExists{true},
		},
		{
			name: "gcp: no org id specified",
			config: Config{
				GCP: GCPConfig{
					PrivateKeyFilename: "/some/valid/path/id_rsa",
				},
			},
			wantErr: true,
		},
		{
			name: "gcp: no private key file specified",
			config: Config{
				GCP: GCPConfig{
					OrgID: "123",
				},
			},
			wantErr: true,
		},
		{
			name: "gcp: private key does not exist",
			config: Config{
				GCP: GCPConfig{
					PrivateKeyFilename: "/some/valid/path/id_rsa",
					OrgID:              "123",
				},
			},
			fileExists: testExists{false},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.fileExists)
			if err != nil && !tt.wantErr {
				t.Fatalf("Unexpected error when validating:\n%v", err)
			}
			if err == nil && tt.wantErr {
				t.Fatalf("Expected error when validating")
			}
		})
	}
}
