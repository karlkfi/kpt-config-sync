package kubectl

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/toolkit/exec"
	"github.com/pkg/errors"
)

func TestClusterList(t *testing.T) {
	// These tests do not need meta client, turn it off.
	useMetaClient = false
	tests := []struct {
		name       string
		configText string
		expected   ClusterList
		err        error
	}{
		{
			name: "Basic",
			expected: ClusterList{
				Clusters: map[string]string{},
				Current:  "",
			},
		},
		{
			name: "OneConfig",
			expected: ClusterList{
				Clusters: map[string]string{
					"dev-frontend": "development",
					"exp-scratch":  "scratch",
				},
				Current: "dev-frontend",
			},
			configText: `` +
				`apiVersion: v1
kind: Config
preferences: {}
clusters:
- cluster:
  name: development
- cluster:
  name: scratch
users:
- name: developer
- name: experimenter
contexts:
- context:
    cluster: development
  name: dev-frontend
- context:
    cluster: scratch
  name: exp-scratch
current-context: dev-frontend
`,
		},
		{
			name: "Unparseable config",
			expected: ClusterList{
				Clusters: map[string]string{
					"dev-frontend": "development",
					"exp-scratch":  "scratch",
				},
				Current: "dev-frontend",
			},
			configText: "the_unparseable_config",
			err:        errors.Errorf("cannot unmarshal string"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TempDir is writable in the build container.
			d, err := ioutil.TempDir("", "home")
			if err != nil {
				t.Fatalf("could not create temp directory: %v", err)
			}
			defer os.Remove(d)
			// Replacement for user.Current() which is not usable without CGO.
			restconfig.SetCurrentUserForTest(
				&user.User{
					Uid:      "0",
					Username: "nobody",
					HomeDir:  filepath.Join(d, "nobody")}, nil)
			err = os.MkdirAll(filepath.Join(d, "nobody/.kube"), os.ModeDir|os.ModePerm)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cfg, err := os.Create(filepath.Join(d, "nobody/.kube/config"))
			if err != nil {
				t.Fatalf("could not open config: %v", err)
			}
			defer os.Remove(cfg.Name())
			fmt.Fprint(cfg, tt.configText)
			err = cfg.Close()
			if err != nil {
				t.Fatalf("could not close config: %v", err)
			}
			cl, err := LocalClusters()
			if err != nil {
				if tt.err != nil {
					if !strings.ContainsAny(tt.err.Error(), err.Error()) {
						t.Errorf("wront error: %q, want: %q", err.Error(), tt.err.Error())
					}
				} else {
					t.Errorf("unexpected error: %v", err)
				}

				return
			}
			if !cmp.Equal(tt.expected, cl) {
				t.Errorf("LocalClusters:()\n%#v,\nwant:\n%#v,\ndiff:\n%v",
					cl, tt.expected, cmp.Diff(cl, tt.expected))
			}
		})
	}
}

func TestVersion(t *testing.T) {
	// These tests do not need meta client, turn it off.
	useMetaClient = false
	// Replacement for user.Current() which is not usable without CGO.
	restconfig.SetCurrentUserForTest(&user.User{Uid: "0", Username: "nobody"}, nil)
	tests := []struct {
		name     string
		output   string
		expected semver.Version
		err      error
	}{
		{
			name: "Simple version",
			output: `{
  "clientVersion": {
    "major": "1",
    "minor": "8",
    "gitVersion": "v1.8.6",
    "gitCommit": "6260bb08c46c31eea6cb538b34a9ceb3e406689c",
    "gitTreeState": "clean",
    "buildDate": "2017-12-21T06:34:11Z",
    "goVersion": "go1.8.3",
    "compiler": "gc",
    "platform": "linux/amd64"
  },
  "serverVersion": {
    "major": "1",
    "minor": "9",
    "gitVersion": "v1.9.1",
    "gitCommit": "3a1c9449a956b6026f075fa3134ff92f7d55f812",
    "gitTreeState": "clean",
    "buildDate": "2018-01-04T11:40:06Z",
    "goVersion": "go1.9.2",
    "compiler": "gc",
    "platform": "linux/amd64"
  }
}
		`,
			expected: semver.MustParse("1.9.1"),
		},
		{
			name: "Complex semver",
			output: `{
  "clientVersion": {
  },
  "serverVersion": {
    "gitVersion": "v1.9.2-rc.alpha.something.other+dirty"
  }
}
		`,
			expected: semver.MustParse("1.9.2-rc.alpha.something.other+dirty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec.SetFakeOutputsForTest(strings.NewReader(tt.output), nil, nil)
			c := New(context.Background())
			actual, err := c.GetClusterVersion()
			if err != nil {
				if tt.err != nil && err.Error() != tt.err.Error() {
					t.Errorf("err.Error(): %v, want: %v", err, tt.err)
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if actual.NE(tt.expected) {
				t.Errorf("actual: %v, want: %v", actual, tt.expected)
			}
		})
	}
}
