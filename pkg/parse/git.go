// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parse

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	v1 "kpt.dev/configsync/pkg/api/configmanagement/v1"
	"kpt.dev/configsync/pkg/hydrate"
	"kpt.dev/configsync/pkg/importer/filesystem/cmpath"
	"kpt.dev/configsync/pkg/importer/git"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/reconcilermanager"
	"kpt.dev/configsync/pkg/status"
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
	// SyncDir is the path to the directory of policies within the source repository.
	SyncDir cmpath.Relative
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

	// currentSyncDir is the directory (including git commit hash) last seen by the Parser.
	currentSyncDir string
}

// gitState contains all state read from the mounted Git repo.
type gitState struct {
	// commit is the Git commit hash read from the Git repo.
	commit string
	// syncDir is the absolute path to the sync directory that includes the configurations.
	syncDir cmpath.Absolute
	// files is the list of all observed files in the sync directory (recursively).
	files []cmpath.Absolute
}

// readConfigFiles reads all the files under state.syncDir and sets state.files.
// - if rendered is true, state.syncDir contains the hydrated files.
// - if rendered is false, state.syncDir contains the source files.
// readConfigFiles should be called after gitState is populated.
func (o *files) readConfigFiles(state *gitState) status.Error {
	if state == nil || state.commit == "" || state.syncDir.OSPath() == "" {
		return status.InternalError("gitState is not populated yet")
	}
	syncDir := state.syncDir
	if syncDir.OSPath() == o.currentSyncDir {
		klog.V(4).Infof("The configs directory is unchanged: %s", syncDir.OSPath())
	} else {
		klog.Infof("Reading updated configs dir: %s", syncDir.OSPath())
		o.currentSyncDir = syncDir.OSPath()
	}

	var fileList []cmpath.Absolute
	var err error
	fileList, err = listFiles(syncDir, map[string]bool{".git": true})
	if err != nil {
		return status.PathWrapError(errors.Wrap(err, "listing files in the configs directory"), syncDir.OSPath())
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

// readHydratedDir returns a gitState object whose `commit` and `syncDir` fields are set if succeeded.
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

	relSyncDir := hydratedDir.Join(o.SyncDir)
	syncDir, err := relSyncDir.EvalSymlinks()
	if err != nil {
		return result, hydrate.NewInternalError(errors.Wrapf(err, "unable to evaluate symbolic link to the hydrated sync directory: %s", relSyncDir.OSPath()))
	}
	result.syncDir = syncDir
	return result, nil
}

// listFiles returns a list of all files in the specified directory.
func listFiles(dir cmpath.Absolute, ignore map[string]bool) ([]cmpath.Absolute, error) {
	var result []cmpath.Absolute
	err := filepath.Walk(dir.OSPath(),
		func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				if _, contains := ignore[fi.Name()]; contains {
					return filepath.SkipDir
				}
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
