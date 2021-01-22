package git

import (
	"testing"
)

func TestCommitHash(t *testing.T) {
	for _, tc := range []struct {
		name    string
		dirPath string
		want    string
		wantErr bool
	}{
		{
			"valid commit hash",
			"/repo/3f8c6da2622fec5896c1e230bda3c53c17f61e8a",
			"3f8c6da2622fec5896c1e230bda3c53c17f61e8a",
			false,
		},
		{
			"invalid length",
			"/repo/abcdef123",
			"",
			true,
		},
		{
			"invalid characters",
			"/repo/zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			"",
			true,
		},
		{
			"more characters after commit hash",
			"/repo/3f8c6da2622fec5896c1e230bda3c53c17f61e8a1111",
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
