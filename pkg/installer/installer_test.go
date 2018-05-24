package installer

import (
	"testing"

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
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			i := &Installer{c: tt.config}
			got := i.getGitConfigMapData()
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("expected %v got %v\ndiff %v", tt.want, got, diff)
			}
		})
	}
}
