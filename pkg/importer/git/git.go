// Package git provides functionality related to Git repos.
package git

import (
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/pkg/errors"
)

// gitSyncPrefix is prepended to each commit hash by git-sync. See:
// https://github.com/kubernetes/git-sync/blob/v2.0.6/cmd/git-sync/main.go#L278
const gitSyncPrefix string = "rev-"

// CommitHash parses the Git commit hash from the given directory path. The format for the path
// should follow the pattern produced by git-sync: /{root}/rev-{hash}
func CommitHash(dirPath string) (string, error) {
	dirName := filepath.Base(dirPath)

	if !strings.HasPrefix(dirName, gitSyncPrefix) {
		return "", errors.Errorf("directory path %q is missing git-sync prefix %q", dirPath, gitSyncPrefix)
	}
	hash := dirName[len(gitSyncPrefix):]
	return hash, nil
}

// CheckClean returns an error if the repo pointed to by dir is not clean, or there was an error invoking Git while
// checking.
func CheckClean(dir string) error {
	cmd := exec.Command("git", "-C", dir, "status", "--short")
	outBytes, err := cmd.CombinedOutput()
	out := string(outBytes)
	if err != nil {
		return errors.Wrapf(err, "checking for clean working directory: failed to call git status on dir %s, output: %s", dir, out)
	}
	if out != "" {
		return errors.Errorf("git status returned dirty working tree: %s", out)
	}
	return nil
}

// ListFiles returns a list of all files tracked by git in the specified
// repo directory.
func ListFiles(dir cmpath.Absolute) ([]cmpath.Absolute, error) {
	out, err := exec.Command("git", "-C", dir.OSPath(), "ls-files").CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, string(out))
	}
	files := strings.Split(string(out), "\n")
	var result []cmpath.Absolute
	// The output from git ls-files, when split on newline, will include an empty string at the end which we don't want.
	for _, f := range files[:len(files)-1] {
		// The result of git ls-files is paths relative to the current working
		// directory, so we have to convert these to absolute paths.
		abs, err := dir.Join(cmpath.RelativeOS(f)).EvalSymlinks()
		if err != nil {
			return nil, errors.Wrapf(err, "error getting absolute path for %q", f)
		}
		result = append(result, abs)
	}

	// git submodule foreach --recursive git ls-files
	out, err = exec.Command("git", "-C", dir.OSPath(), "submodule",
		// Execute the command for every submodule.
		"foreach",
		// Recursively execute the command for every submodule's submodule.
		"--recursive",
		// List the files in the submodule.
		"git", "ls-files").
		CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, string(out))
	}

	lines := strings.Split(string(out), "\n")
	prefix := ""
	for _, l := range lines {
		if len(l) == 0 {
			// If the repo has no submodules, lines is just [""] and we want to ignore that.
			continue
		}
		// Before entering a submodule, git prints the path from the root of the
		// repository to the submodule. For example:
		// Entering 'namespaces/foo'
		if strings.HasPrefix(l, "Entering '") {
			prefix = l[10 : len(l)-1]
			continue
		}
		abs := dir.Join(cmpath.RelativeOS(prefix)).Join(cmpath.RelativeOS(l))
		result = append(result, abs)
	}

	return result, nil
}
