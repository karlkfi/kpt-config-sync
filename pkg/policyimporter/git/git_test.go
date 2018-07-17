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
			"repo/rev-abcdef123/my-policies",
			"abcdef123",
			false,
		},
		{
			"Missing git-sync prefix",
			"abcdef123/my-policies",
			"",
			true,
		},
		{
			"Missing policy dir suffix",
			"repo/rev-abcdef123",
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
