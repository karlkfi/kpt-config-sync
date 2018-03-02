package kubectl

import (
	"context"
	"os/user"
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/google/stolos/pkg/toolkit/exec"
)

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
