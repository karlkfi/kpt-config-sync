package installer

import (
	"testing"

	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/installer/config"
)

func TestGitConfigMap(t *testing.T) {
	testCases := []struct {
		name   string
		config config.Config
		want   []string
	}{
		{
			name: "git ssh",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo:           "git@github.com:user/foo-corp.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/path/id_rsa",
					KnownHostsFilename: "/some/path/known_hosts",
					SyncBranch:         "master",
					RootPolicyDir:      "foo-corp",
					SyncWaitSeconds:    60,
				},
			},
			want: []string{
				"GIT_SYNC_SSH=true",
				"GIT_SYNC_REPO=git@github.com:user/foo-corp.git",
				"GIT_SYNC_BRANCH=master",
				"GIT_SYNC_WAIT=60",
				"GIT_KNOWN_HOSTS=true",
				"GIT_COOKIE_FILE=false",
				"POLICY_DIR=foo-corp",
			},
		},
		{
			name: "git ssh, no known hosts",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo:           "git@github.com:user/foo-corp.git",
					UseSSH:             true,
					PrivateKeyFilename: "/some/path/id_rsa",
					SyncBranch:         "master",
					RootPolicyDir:      "foo-corp",
					SyncWaitSeconds:    60,
				},
			},
			want: []string{
				"GIT_SYNC_SSH=true",
				"GIT_SYNC_REPO=git@github.com:user/foo-corp.git",
				"GIT_SYNC_BRANCH=master",
				"GIT_SYNC_WAIT=60",
				"GIT_KNOWN_HOSTS=false",
				"GIT_COOKIE_FILE=false",
				"POLICY_DIR=foo-corp",
			},
		},
		{
			name: "git https",
			config: config.Config{
				Git: config.GitConfig{
					SyncRepo:        "https://github.com/sbochins-k8s/foo-corp-example.git",
					UseSSH:          false,
					SyncBranch:      "master",
					RootPolicyDir:   "foo-corp",
					SyncWaitSeconds: 60,
					CookieFilename:  "~/.gitcookies",
				},
			},
			want: []string{
				"GIT_SYNC_SSH=false",
				"GIT_SYNC_REPO=https://github.com/sbochins-k8s/foo-corp-example.git",
				"GIT_SYNC_BRANCH=master",
				"GIT_SYNC_WAIT=60",
				"GIT_KNOWN_HOSTS=false",
				"GIT_COOKIE_FILE=true",
				"POLICY_DIR=foo-corp",
			},
		},
		{
			name: "gcp",
			config: config.Config{
				GCP: config.GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/some/file",
				},
			},
			want: []string{
				"ORG_ID=1234",
			},
		},
		{
			name: "gcp with api address",
			config: config.Config{
				GCP: config.GCPConfig{
					OrgID:              "1234",
					PrivateKeyFilename: "/some/file",
					PolicyAPIAddress:   "localhost:1234",
				},
			},
			want: []string{
				"ORG_ID=1234",
				"POLICY_API_ADDRESS=localhost:1234",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			i := &Installer{c: tt.config}
			var got []string
			if strings.Contains(tt.name, "git") {
				got = i.gitConfigMapContent()
			} else if strings.Contains(tt.name, "gcp") {
				got = i.gcpConfigMapContent()
			} else {
				t.Errorf("test case name must contain either git or gcp")
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("expected %v got %v\ndiff %v", tt.want, got, diff)
			}
		})
	}
}
