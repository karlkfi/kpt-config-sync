package parse

import (
	"math"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/status"
)

const (
	retriesBeforeStartingBackoff = 5
	maxRetryInterval             = time.Duration(5) * time.Minute
)

type gitStatus struct {
	commit string
	errs   status.MultiError
}

func (gs gitStatus) equal(other gitStatus) bool {
	return gs.commit == other.commit && status.DeepEqual(gs.errs, other.errs)
}

type renderingStatus struct {
	commit  string
	message string
	errs    status.MultiError
}

func (rs renderingStatus) equal(other renderingStatus) bool {
	return rs.commit == other.commit && rs.message == other.message && status.DeepEqual(rs.errs, other.errs)
}

type reconcilerState struct {
	// lastApplied keeps the state for the last successful-applied policyDir.
	lastApplied string

	// sourceStatus tracks info from the `Status.Source` field of a RepoSync/RootSync.
	sourceStatus gitStatus

	// renderingStatus tracks info from the `Status.Rendering` field of a RepoSync/RootSync.
	renderingStatus renderingStatus

	// syncStatus tracks info from the `Status.Sync` field of a RepoSync/RootSync.
	syncStatus gitStatus

	// cache tracks the progress made by the reconciler for a git commit.
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
	s.cache.reconciliationWithSameErrs = 0
	s.cache.nextRetryTime = time.Time{}
	s.cache.errs = nil
}

// invalidate logs the errors, clears the state tracking information.
// invalidate does not clean up the `s.cache`.
func (s *reconcilerState) invalidate(errs status.MultiError) {
	glog.Errorf("Invalidating reconciler checkpoint: %v", status.FormatSingleLine(errs))
	oldErrs := s.cache.errs
	s.cache.errs = errs
	// Invalidate state on error since this could be the result of switching
	// branches or some other operation where inverting the operation would
	// result in repeating a previous state that was checkpointed.
	s.lastApplied = ""
	s.cache.needToRetry = true
	if status.DeepEqual(oldErrs, s.cache.errs) {
		s.cache.reconciliationWithSameErrs++
	} else {
		s.cache.reconciliationWithSameErrs = 1
	}
	s.cache.nextRetryTime = calculateNextRetryTime(s.cache.reconciliationWithSameErrs)
}

func calculateNextRetryTime(retries int) time.Time {
	// For the first several retries, the reconciler waits 1 second before retrying.
	if retries <= retriesBeforeStartingBackoff {
		return time.Now().Add(time.Second)
	}

	// For the remaining retries, the reconciler does exponential backoff retry up to 5 minutes.
	// i.e., 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 256s, 5m, 5m, ...
	seconds := int64(math.Pow(2, float64(retries-retriesBeforeStartingBackoff)))
	duration := time.Duration(seconds) * time.Second
	if duration > maxRetryInterval {
		duration = maxRetryInterval
	}
	return time.Now().Add(duration)
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
