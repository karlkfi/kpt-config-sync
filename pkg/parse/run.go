package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
)

const (
	triggerResync             = "resync"
	triggerReimport           = "reimport"
	triggerRetry              = "retry"
	triggerManagementConflict = "managementConflict"
	triggerWatchUpdate        = "watchUpdate"
)

// Run keeps checking whether a parse-apply-watch loop is necessary and starts a loop if needed.
func Run(ctx context.Context, p Parser) {
	tickerPoll := time.NewTicker(p.getPollingFrequency())
	tickerResync := time.NewTicker(p.getResyncPeriod())
	tickerRetryOrWatchUpdate := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return

		// it is time to reapply the configuration even if no changes have been detected
		// This case should be checked first since it resets the cache
		case <-tickerResync.C:
			glog.Infof("It is time for a force-resync")
			// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
			p.resetCache()
			readAndParse(ctx, p, triggerResync)

		// it is time to re-import the configuration from the filesystem
		case <-tickerPoll.C:
			readAndParse(ctx, p, triggerReimport)

		// it is time to check whether the last parse-apply-watch loop failed or any watches need to be updated
		case <-tickerRetryOrWatchUpdate.C:
			var trigger string
			if p.managementConflict() {
				glog.Infof("One of the watchers noticed a management conflict")
				// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
				p.resetCache()
				trigger = triggerManagementConflict
			} else if p.getCache().needToRetry {
				glog.Infof("The last parse-apply-watch loop failed")
				trigger = triggerRetry
			} else if p.needToUpdateWatch() {
				glog.Infof("Some watches need to be updated")
				trigger = triggerWatchUpdate
			} else {
				continue
			}
			readAndParse(ctx, p, trigger)
		}
	}
}

func readAndParse(ctx context.Context, p Parser, trigger string) {
	start := time.Now()
	errs := read(ctx, p)
	metrics.RecordParserDuration(ctx, trigger, "read", metrics.StatusTagKey(errs), start)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		p.invalidate(errs)
		return
	}

	start = time.Now()
	errs = parse(ctx, p)
	metrics.RecordParserDuration(ctx, trigger, "parse", metrics.StatusTagKey(errs), start)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		p.invalidate(errs)
	}
}

// read reads the git commit and policyDir from the git repo, checks whether the gitstate in
// the cache is up-to-date. If the cache is not up-to-date, reads all the git files from the
// git repo and caches the gitstate.
//
// read does not log any error encountered, instead returns them to its caller.
func read(ctx context.Context, p Parser) status.MultiError {
	state, err := p.readGitCommitAndPolicyDir(p.getReconcilerName())
	if err != nil {
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			return status.Append(err, err2)
		}
		return err
	}

	if state.policyDir == p.getCache().git.policyDir {
		return nil
	}

	glog.Infof("New git changes (%s) detected, reset the cache", state.policyDir.OSPath())

	// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
	p.resetCache()

	// Read all the files under state.policyDir
	if err := p.readGitFiles(&state); err != nil {
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			return status.Append(err, err2)
		}
		return err
	}

	// Set p.getCache().git after p.readGitFiles succeeded
	p.getCache().git = state
	return nil
}

// parse implements the parse-apply-watch loop.
//
// parse does not log any error encountered, instead returns them to its caller.
func parse(ctx context.Context, p Parser) status.MultiError {
	// Parse the declared resources
	var cos []core.Object
	var sourceErrs, syncErrs status.MultiError
	var setSourceStatusErr, setSyncStatusErr error

	gitState := p.getCache().git

	if p.getCache().hasParserResult {
		cos = p.getCache().parserResult
	} else {
		cos, sourceErrs = p.parseSource(ctx, gitState)
		if sourceErrs == nil {
			p.getCache().setParserResult(cos)
		}
	}

	if !p.getCache().sourceStatusUpdated {
		setSourceStatusErr = p.setSourceStatus(ctx, gitState, sourceErrs)
		if setSourceStatusErr == nil {
			p.getCache().sourceStatusUpdated = true
		}
		if sourceErrs != nil {
			return status.Append(sourceErrs, setSourceStatusErr)
		}
	}

	syncErrs = p.update(ctx, cos)
	if syncErrs != nil || !p.getCache().syncStatusUpdated {
		if setSyncStatusErr = p.setSyncStatus(ctx, gitState.commit, syncErrs); setSyncStatusErr != nil {
			p.getCache().syncStatusUpdated = false
		} else {
			p.getCache().syncStatusUpdated = true
		}
	}

	// Only checkpoint our state *everything* succeeded, including status update.
	if sourceErrs == nil && syncErrs == nil && setSourceStatusErr == nil && setSyncStatusErr == nil {
		p.checkpoint(gitState.policyDir.OSPath())
		return nil
	}

	return status.Append(sourceErrs, syncErrs, setSourceStatusErr, setSyncStatusErr)
}
