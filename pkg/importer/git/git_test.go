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
			"Valid path",
			"/repo/rev-abcdef123/my-policies",
			"abcdef123",
			false,
		},
		{
			"Valid path nested",
			"/repo/rev-abcdef123/path/to/my/policies",
			"abcdef123",
			false,
		},
		{
			"Valid path root",
			"/repo/rev-abcdef123",
			"abcdef123",
			false,
		},
		{
			"Invalid path",
			"/abcdef123",
			"",
			true,
		},
		{
			"Missing git-sync prefix",
			"/repo/abdef123/my-policies",
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
