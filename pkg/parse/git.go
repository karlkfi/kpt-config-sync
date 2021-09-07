package parse

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

// FileSource includes all settings to configure where a Parser reads files from.
type FileSource struct {
	// GitDir is the path to the symbolic link of the git repository.
	GitDir cmpath.Absolute
	// HydratedRoot is the path to the root of the hydrated directory.
	HydratedRoot string
	// RepoRoot is the absolute path to the parent directory of GitRoot and HydratedRoot.
	RepoRoot cmpath.Absolute
	// HydratedLink is the relative path to the symbolic link of the hydrated configs.
	HydratedLink string
	// PolicyDir is the path to the directory of policies within the git repository.
	PolicyDir cmpath.Relative
	// GitRepo is the git repo to sync.
	GitRepo string
	// GitBranch is the branch of the git repo to sync.
	GitBranch string
	// GitRev is the revision of the git repo to sync.
	GitRev string
}

// files lists files in a repository and ensures the Git repository hasn't been
// modified from HEAD.
type files struct {
	FileSource

	// currentPolicyDir is the directory (including git commit hash) last seen by the Parser.
	currentPolicyDir string
}

// gitState contains all state read from the mounted Git repo.
type gitState struct {
	// commit is the Git commit hash read from the Git repo.
	commit string
	// policyDir is the absolute path to the policy directory.
	policyDir cmpath.Absolute
	// files is the list of all observed files in the policy directory (recursively).
	files []cmpath.Absolute
}

// readConfigFiles reads all the files under state.policyDir and sets state.files.
// - if rendered is true, state.policyDir contains the hydrated files.
// - if rendered is false, state.policyDir contains the source files.
// readConfigFiles should be called after gitState is populated.
func (o *files) readConfigFiles(state *gitState, rendered bool) status.Error {
	if state == nil || state.commit == "" || state.policyDir.OSPath() == "" {
		return status.InternalError("gitState is not populated yet")
	}
	policyDir := state.policyDir
	if policyDir.OSPath() == o.currentPolicyDir {
		glog.V(4).Infof("The configs directory is unchanged: %s", policyDir.OSPath())
	} else {
		glog.Infof("Reading updated configs dir: %s", policyDir.OSPath())
		o.currentPolicyDir = policyDir.OSPath()
	}

	var fileList []cmpath.Absolute
	var err error
	if rendered {
		fileList, err = listFiles(policyDir)
	} else {
		fileList, err = git.ListFiles(policyDir)
	}
	if err != nil {
		return status.PathWrapError(errors.Wrap(err, "listing files in the configs directory"), policyDir.OSPath())
	}
	state.files = fileList
	return nil
}

func (o *files) gitContext() gitContext {
	return gitContext{
		Repo:   o.GitRepo,
		Branch: o.GitBranch,
		Rev:    o.GitRev,
	}
}

// readHydratedDir returns a gitState object whose `commit` and `policyDir` fields are set if succeeded.
func (o *files) readHydratedDir(hydratedRoot cmpath.Absolute, link, reconciler string) (gitState, hydrate.HydrationError) {
	result := gitState{}
	errorFile := hydratedRoot.Join(cmpath.RelativeSlash(hydrate.ErrorFile))
	if _, err := os.Stat(errorFile.OSPath()); err == nil {
		return result, hydratedError(errorFile.OSPath(),
			fmt.Sprintf("%s=%s", metadata.ReconcilerLabel, reconciler))
	} else if !os.IsNotExist(err) {
		return result, hydrate.NewInternalError(errors.Wrapf(err, "failed to check the error file: %s", errorFile.OSPath()))
	}
	hydratedDir, err := hydratedRoot.Join(cmpath.RelativeSlash(link)).EvalSymlinks()
	if err != nil {
		return result, hydrate.NewInternalError(errors.Wrapf(err, "unable to load the hydrated configs under %s", hydratedRoot.OSPath()))
	}

	commit, err := git.CommitHash(hydratedDir.OSPath())
	if err != nil {
		return result, hydrate.NewInternalError(errors.Wrapf(err, "unable to parse commit hash from the hydrated directory: %s", hydratedDir.OSPath()))
	}
	result.commit = commit

	relSyncDir := hydratedDir.Join(o.PolicyDir)
	syncDir, err := relSyncDir.EvalSymlinks()
	if err != nil {
		return result, hydrate.NewInternalError(errors.Wrapf(err, "unable to evaluate symbolic link to the hydrated sync directory: %s", relSyncDir.OSPath()))
	}
	result.policyDir = syncDir
	return result, nil
}

// listFiles returns a list of all files in the specified directory.
func listFiles(dir cmpath.Absolute) ([]cmpath.Absolute, error) {
	var result []cmpath.Absolute
	err := filepath.Walk(dir.OSPath(),
		func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			abs, err := cmpath.AbsoluteOS(path)
			if err != nil {
				return err
			}
			abs, err = abs.EvalSymlinks()
			if err != nil {
				return err
			}
			result = append(result, abs)
			return nil
		})
	return result, err
}

// hydratedError returns the error details from the error file generated by the hydration controller.
func hydratedError(errorFile, label string) hydrate.HydrationError {
	content, err := ioutil.ReadFile(errorFile)
	if err != nil {
		return hydrate.NewInternalError(errors.Errorf("Unable to load %s: %v. Please check %s logs for more info: kubectl logs -n %s -l %s -c %s",
			errorFile, err, reconcilermanager.HydrationController, v1.NSConfigManagementSystem, label, reconcilermanager.HydrationController))
	}
	if len(content) == 0 {
		return hydrate.NewInternalError(fmt.Errorf("%s is empty. Please check %s logs for more info: kubectl logs -n %s -l %s -c %s",
			errorFile, reconcilermanager.HydrationController, v1.NSConfigManagementSystem, label, reconcilermanager.HydrationController))
	}

	payload := &hydrate.HydrationErrorPayload{}
	if err := json.Unmarshal(content, payload); err != nil {
		return hydrate.NewInternalError(err)
	}
	if payload.Code == status.ActionableHydrationErrorCode {
		return hydrate.NewActionableError(errors.New(payload.Error))
	}
	return hydrate.NewInternalError(errors.New(payload.Error))
}
