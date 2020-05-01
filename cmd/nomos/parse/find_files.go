package parse

import (
	"os/exec"
	"strings"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/pkg/errors"
)

// FindFiles lists what are likely the files tracked by git in cases where
// we may not be dealing with a git repository. ONLY FOR USE IN THE CLI.
//
// Tries git first, and falls back to using `find` if git does not work.
//
// Guaranteed to return the same files as ListFiles in git repo with no
// uncommitted changes (see tests for findFiles)
func FindFiles(dir cmpath.Path) ([]cmpath.Path, error) {
	out, err := exec.Command("find", dir.OSPath(), "-type", "f", "-not", "-path", "*/\\.git/*").CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, string(out))
	}
	files := strings.Split(string(out), "\n")
	var result []cmpath.Path
	// The output from git ls-files, when split on newline, will include an empty string at the end which we don't want.
	for _, f := range files[:len(files)-1] {
		result = append(result, cmpath.FromOS(f))
	}
	return result, nil
}
