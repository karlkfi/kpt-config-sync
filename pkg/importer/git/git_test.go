package git

import "testing"

func TestCommitHash(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dirPath string
		want    string
		wantErr bool
	}{
		{
			"Invalid path nested",
			"/repo/rev-abcdef123/path/to/my/configs",
			"",
			true,
		},
		{
			"Valid path root",
			"/repo/rev-abcdef123",
			"abcdef123",
			false,
		},
		{
			"Pathological (no pun intended) invalid stuff before rev-",
			"/repo/pathological-dirname-rev-abcdef123",
			"",
			true,
		},
		{
			"Invalid path",
			"/abcdef123",
			"",
			true,
		},
		{
			"Missing git-sync prefix",
			"/repo/abdef123/my-configs",
			"",
			true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CommitHash(tc.dirPath)
			if tc.wantErr {
				if err == nil {
					t.Errorf("CommitHash(%q) got nil error, want error", tc.dirPath)
				}
			} else if err != nil {
				t.Errorf("CommitHash(%q) got error %v, want nil error", tc.dirPath, err)
			}
			if got != tc.want {
				t.Errorf("CommitHash(%q) got %q, want %q", tc.dirPath, got, tc.want)
			}
		})
	}
}
