package configgen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/google/nomos/pkg/process/dialog"
	"github.com/google/nomos/pkg/process/exec"
)

func TestGitForm(t *testing.T) {
	tests := []struct {
		name string
		// output format is whatever is in the git form, one line per
		// entry, in the order initialized in the form itself.
		output string
		// expected is the resulting git configuration that resulted from the
		// edits.
		expected config.GitConfig
		// err is the error expected at end.
		err error
	}{
		{
			name: "Basic",
			output: `git@repo.com:foo/bar.git
Y
private_key_filename
known_hosts_filename
sync_branch
root_policy_dir
666
`,
			expected: config.GitConfig{
				SyncRepo:           "git@repo.com:foo/bar.git",
				UseSSH:             true,
				PrivateKeyFilename: "private_key_filename",
				KnownHostsFilename: "known_hosts_filename",
				SyncBranch:         "sync_branch",
				RootPolicyDir:      "root_policy_dir",
				SyncWaitSeconds:    666,
			},
		},
		{
			name: "Do not use SSH",
			output: `git@repo.com:foo/bar.git
n
private_key_filename
known_hosts_filename
sync_branch
root_policy_dir
666
`,
			expected: config.GitConfig{
				SyncRepo:           "git@repo.com:foo/bar.git",
				UseSSH:             false,
				PrivateKeyFilename: "private_key_filename",
				KnownHostsFilename: "known_hosts_filename",
				SyncBranch:         "sync_branch",
				RootPolicyDir:      "root_policy_dir",
				SyncWaitSeconds:    666,
			},
		},
		{
			name: "Unparseable 'use ssh'",
			output: `git@repo.com:foo/bar.git
NNNNOOOO!!!1!
private_key_filename
known_hosts_filename
sync_branch
root_policy_dir
666
`,
			err: fmt.Errorf("while filling in git config form must specify Y or n for GIT_SYNC_SSH field"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exec.SetFakeOutputsForTest(nil, strings.NewReader(test.output), nil)
			cfg := config.GitConfig{}
			f := NewGitForm(dialog.NewOptions(), &cfg)
			retry, err := f.Run()
			if err != nil {
				if test.err == nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if retry {
				t.Errorf("retry=%v, want false", retry)
			}
			if !cmp.Equal(cfg, test.expected) {
				t.Errorf("cfg=%+v, want: %+v, diff:\n%v",
					cfg, test.expected, cmp.Diff(cfg, test.expected))
			}
		})
	}
}
