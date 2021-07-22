package hydrate

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/git"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

const (
	// tmpLink is the temporary soft link name.
	tmpLink = "tmp-link"
	// DoneFile is the file name that indicates the hydration is done.
	DoneFile = "done"
	// ErrorFile is the file name of the hydration errors.
	ErrorFile = "error.txt"
)

// Hydrator runs the hydration process.
type Hydrator struct {
	// DonePath is the absolute path to the done file under the /repo directory.
	DonePath cmpath.Absolute
	// SourceRoot is the absolute path to the source root directory.
	SourceRoot cmpath.Absolute
	// HydratedRoot is the absolute path to the hydrated root directory.
	HydratedRoot cmpath.Absolute
	// SourceLink is the name of (a symlink to) the source directory under SourceRoot, which contains the clone of the git repo.
	SourceLink string
	// HydratedLink is the name of (a symlink to) the source directory under HydratedRoot, which contains the hydrated configs.
	HydratedLink string
	// SyncDir is the relative path to the configs within the Git repository.
	SyncDir cmpath.Relative
	// PollingFrequency is the period of time between checking the filesystem for rendering the DRY configs.
	PollingFrequency time.Duration
	// ReconcilerName is the name of the reconciler.
	ReconcilerName string
}

// Run runs the hydration process periodically.
func (h *Hydrator) Run(ctx context.Context) {
	tickerPoll := time.NewTicker(h.PollingFrequency)
	absSourceDir := h.SourceRoot.Join(cmpath.RelativeSlash(h.SourceLink))
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerPoll.C:
			commit, syncDir, err := SourceCommitAndDir(absSourceDir, h.SyncDir, h.ReconcilerName)
			if err != nil {
				glog.Errorf("failed to get the commit hash and sync directory from the source directory %s: %v", absSourceDir.OSPath(), err)
			} else {
				hydrateErr := h.hydrate(commit, syncDir.OSPath())
				if err := h.complete(commit, hydrateErr); err != nil {
					glog.Errorf("failed to complete the rendering execution for commit %q: %v", commit, err)
				}
			}
		}
	}
}

// hydrate renders the source git repo to hydrated configs.
func (h *Hydrator) hydrate(sourceCommit, syncDir string) error {
	hydrate, err := NeedsKustomize(syncDir)
	if err != nil {
		return errors.Wrapf(err, "unable to check if rendering is needed for the source directory: %s", syncDir)
	}
	if !hydrate {
		glog.V(5).Infof("no rendering is needed because of no Kustomization config file in the source configs with commit %s", sourceCommit)
		return os.RemoveAll(h.HydratedRoot.OSPath())
	}
	hydratedCommit := ""
	absHydratedDir := h.HydratedRoot.Join(cmpath.RelativeSlash(h.HydratedLink))
	hydratedDir, err := absHydratedDir.EvalSymlinks()
	if err == nil {
		hydratedCommit, err = git.CommitHash(hydratedDir.OSPath())
		if err != nil {
			return errors.Wrapf(err, "unable to parse commit hash from the hydrated directory: %s", hydratedDir.OSPath())
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "unable to evaluate the hydrated directory: %s", absHydratedDir.OSPath())
	}

	// no hydration is needed because there is no change from the source.
	if hydratedCommit == sourceCommit {
		return nil
	}

	// Remove the done file because a new hydration is in progress.
	if err := os.RemoveAll(h.DonePath.OSPath()); err != nil {
		return errors.Wrapf(err, "unable to remove the done file: %s", h.DonePath.OSPath())
	}

	newHydratedDir := h.HydratedRoot.Join(cmpath.RelativeOS(sourceCommit))
	dest := newHydratedDir.Join(h.SyncDir).OSPath()

	if err := KustomizeBuild(syncDir, dest); err != nil {
		return errors.Wrapf(err, "unable to render the source configs in %s", syncDir)
	}
	if err := updateSymlink(h.HydratedRoot.OSPath(), h.HydratedLink, newHydratedDir.OSPath()); err != nil {
		return errors.Wrapf(err, "unable to update the symbolic link to %s", newHydratedDir.OSPath())
	}
	// The Helm inflator might pull remote charts locally, so remove the local charts to make the source directory clean.
	if err := git.ForceClean(syncDir); err != nil {
		return err
	}
	glog.Infof("Successfully rendered %s for commit %s", syncDir, sourceCommit)
	return nil
}

// updateSymlink updates the symbolic link to the hydrated directory.
func updateSymlink(hydratedRoot, link, newDir string) error {
	linkPath := filepath.Join(hydratedRoot, link)
	oldDir, err := filepath.EvalSymlinks(linkPath)
	deleteOldDir := true
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "unable to access the current hydrated directory: %s", linkPath)
		}
		deleteOldDir = false
	}

	// newDir is absolute, so we need to change it to a relative path.  This is
	// so it can be volume-mounted at another path and the symlink still works.
	newDirRelative, err := filepath.Rel(hydratedRoot, newDir)
	if err != nil {
		return errors.Wrap(err, "unable to convert to relative path")
	}

	if _, err := runCommand(hydratedRoot, "ln", "-snf", newDirRelative, tmpLink); err != nil {
		return errors.Wrap(err, "unable to create symlink")
	}

	if _, err := runCommand(hydratedRoot, "mv", "-T", tmpLink, link); err != nil {
		return errors.Wrap(err, "unable to replace symlink")
	}

	if deleteOldDir {
		if err := os.RemoveAll(oldDir); err != nil {
			glog.Warningf("unable to remove the previously hydrated directory %s: %v", oldDir, err)
		}
	}
	return nil
}

// complete marks the hydration process is done with a done file under the /repo directory
// and reset the error file (create, update or delete).
func (h *Hydrator) complete(commit string, hydrationErr error) error {
	errorPath := h.HydratedRoot.Join(cmpath.RelativeSlash(ErrorFile)).OSPath()
	var err error
	if hydrationErr == nil {
		err = deleteErrorFile(errorPath)
	} else {
		err = exportError(commit, h.HydratedRoot.OSPath(), errorPath, hydrationErr)
	}
	if err != nil {
		return err
	}
	done, err := os.Create(h.DonePath.OSPath())
	if err != nil {
		return errors.Wrapf(err, "unable to create done file: %s", h.DonePath.OSPath())
	}
	if err := done.Close(); err != nil {
		glog.Warningf("unable to close the done file %s: %v", h.DonePath.OSPath(), err)
	}
	glog.Infof("Successfully completed rendering execution for commit %s", commit)
	return nil
}

// exportError writes the error content to the error file.
func exportError(commit, root, errorFile string, hydrationError error) error {
	glog.Errorf("rendering error for commit %s: %v", commit, hydrationError)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		fileMode := os.FileMode(0755)
		if err := os.Mkdir(root, fileMode); err != nil {
			return errors.Wrapf(err, "unable to create the root directory: %s", root)
		}
	}

	tmpFile, err := ioutil.TempFile(root, "tmp-err-")
	if err != nil {
		return errors.Wrapf(err, "unable to create temporary error-file under directory %s", root)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			glog.Warningf("unable to close temporary error-file: %s", tmpFile.Name())
		}
	}()

	if _, err = tmpFile.WriteString(hydrationError.Error()); err != nil {
		return errors.Wrapf(err, "unable to write to temporary error-file: %s", tmpFile.Name())
	}
	if err := os.Rename(tmpFile.Name(), errorFile); err != nil {
		return errors.Wrapf(err, "unable to rename %s to %s", tmpFile.Name(), errorFile)
	}
	if err := os.Chmod(errorFile, 0644); err != nil {
		return errors.Wrapf(err, "unable to change permissions on the error-file: %s", errorFile)
	}
	glog.Infof("Saved the rendering error in file: %s", errorFile)
	return nil
}

// deleteErrorFile deletes the error file.
func deleteErrorFile(file string) error {
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "unable to delete error file: %s", file)
	}
	return nil
}

// SourceCommitAndDir returns the source commit hash, the absolute path of the sync directory, and source errors.
func SourceCommitAndDir(sourceRoot cmpath.Absolute, syncDir cmpath.Relative, reconcilerName string) (string, cmpath.Absolute, status.Error) {
	// Check if the source configs are synced successfully.
	errFilePath := filepath.Join(path.Dir(sourceRoot.OSPath()), git.ErrorFile)

	// A function that turns an error to a status sourceError.
	toSourceError := func(err error) status.Error {
		if err == nil {
			err = errors.Errorf("unable to sync repo\n%s",
				git.SyncError(errFilePath, fmt.Sprintf("%s=%s", metadata.ReconcilerLabel, reconcilerName)))
		} else {
			err = errors.Wrapf(err, "unable to sync repo\n%s",
				git.SyncError(errFilePath, fmt.Sprintf("%s=%s", metadata.ReconcilerLabel, reconcilerName)))
		}
		return status.SourceError.Wrap(err).Build()
	}

	if _, err := os.Stat(errFilePath); err == nil || !os.IsNotExist(err) {
		return "", cmpath.Absolute{}, toSourceError(err)
	}
	gitDir, err := sourceRoot.EvalSymlinks()
	if err != nil {
		return "", cmpath.Absolute{}, toSourceError(err)
	}

	commit, e := git.CommitHash(gitDir.OSPath())
	if e != nil {
		return "", cmpath.Absolute{}, status.SourceError.Wrap(e).Sprintf("unable to parse commit hash from source path: %s", gitDir.OSPath()).Build()
	}

	err = git.CheckClean(gitDir.OSPath())
	if err != nil {
		return commit, cmpath.Absolute{}, status.PathWrapError(
			errors.Wrap(err, "checking that the git repository has no changes"), sourceRoot.OSPath())
	}

	relSyncDir := gitDir.Join(syncDir)
	sourceDir, err := relSyncDir.EvalSymlinks()
	if err != nil {
		return commit, cmpath.Absolute{}, status.PathWrapError(
			errors.Wrap(err, "evaluating symbolic link to policy sourceRoot"), relSyncDir.OSPath())
	}
	return commit, sourceDir, nil
}
