package parse

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/status"
)

type gitStatus struct {
	commit string
	errs   status.MultiError
}

func (gs gitStatus) equal(other gitStatus) bool {
	return gs.commit == other.commit && status.DeepEqual(gs.errs, other.errs)
}

type reconcilerState struct {
	// lastApplied keeps the state for the last successful-applied policyDir.
	lastApplied string

	// sourceStatus tracks info from the `Status.Source` field of a RepoSync/RootSync.
	sourceStatus gitStatus

	// syncStatus tracks info from the `Status.Sync` field of a RepoSync/RootSync.
	syncStatus gitStatus

	// cache tracks the progress made by the reconciler for a git commit
	cache cacheForCommit
}

func (s *reconcilerState) checkpoint() {
	applied := s.cache.git.policyDir.OSPath()
	if applied == s.lastApplied {
		return
	}
	glog.Infof("Reconciler checkpoint updated to %s", applied)
	s.cache.errs = nil
	s.lastApplied = applied
	s.cache.needToRetry = false
}

// invalidate logs the errors, clears the state tracking information.
// invalidate does not clean up the `s.cache`.
func (s *reconcilerState) invalidate(err status.MultiError) {
	glog.Errorf("Reconciler checkpoint invalidated with errors: %v", err)
	s.cache.errs = err
	// Invalidate state on error since this could be the result of switching
	// branches or some other operation where inverting the operation would
	// result in repeating a previous state that was checkpointed.
	s.lastApplied = ""
	s.cache.needToRetry = true
}

// resetCache resets the whole cache.
//
// resetCache is called when a new git commit is detected.
func (s *reconcilerState) resetCache() {
	s.cache = cacheForCommit{}
}

// resetAllButGitState resets the whole cache except for the cached gitState.
//
// resetAllButGitState is called when:
//   * a force-resync happens, or
//   * one of the watchers noticed a management conflict.
func (s *reconcilerState) resetAllButGitState() {
	git := s.cache.git
	s.cache = cacheForCommit{}
	s.cache.git = git
}
