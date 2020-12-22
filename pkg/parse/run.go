package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
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
			readAndParse(ctx, p)

		// it is time to re-import the configuration from the filesystem
		case <-tickerPoll.C:
			readAndParse(ctx, p)

		// it is time to check whether the last parse-apply-watch loop failed or any watches need to be updated
		case <-tickerRetryOrWatchUpdate.C:
			if p.managementConflict() {
				glog.Infof("One of the watchers noticed a management conflict")
				// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
				p.resetCache()
			} else if p.getCache().needToRetry {
				glog.Infof("The last parse-apply-watch loop failed")
			} else if p.needToUpdateWatch() {
				glog.Infof("Some watches need to be updated")
			} else {
				continue
			}
			readAndParse(ctx, p)
		}
	}
}

func readAndParse(ctx context.Context, p Parser) {
	state, errs := read(ctx, p)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		p.invalidate(errs)
		return
	}

	errs = parse(ctx, state, p)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		p.invalidate(errs)
	}
}

func read(ctx context.Context, p Parser) (*gitState, status.MultiError) {
	state, err := p.readGitState(p.getReconcilerName())
	if err != nil {
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			glog.Errorf("Failed to set source status: %v", err2)
			return nil, status.Append(err, err2)
		}
		return nil, err
	}
	return &state, nil
}

func setSourceStatus(ctx context.Context, state gitState, p Parser, sourceErrs status.MultiError) error {
	err := p.setSourceStatus(ctx, state, sourceErrs)
	if err != nil {
		glog.Errorf("Failed to set source status: %v", err)
		p.getCache().sourceStatusUpdated = false
	} else {
		p.getCache().sourceStatusUpdated = true
	}
	return err
}

func parse(ctx context.Context, state *gitState, p Parser) status.MultiError {
	if state.policyDir.OSPath() != p.getCache().policyDir {
		glog.Infof("New git changes (%s) detected, reset the cache", state.policyDir.OSPath())
		// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
		p.resetCache()
	}

	// Parse the declared resources
	var cos []core.Object
	var sourceErrs status.MultiError
	if p.getCache().hasParserResult {
		cos = p.getCache().parserResult
	} else {
		cos, sourceErrs = p.parseSource(ctx, state)
		if sourceErrs == nil {
			p.getCache().setParserResult(state.policyDir.OSPath(), cos)
		}
	}

	if !p.getCache().sourceStatusUpdated {
		// At this point, `sourceErrs` only include the errors returned from `p.parseSource`.
		// `setSourceStatus` needs to be called no matter `sourceErrs` are empty or not.
		// Errors from `p.parseSource` will cause this function to return; however,
		// errors from `setSourceStatus` will not.
		err := setSourceStatus(ctx, *state, p, sourceErrs)
		if sourceErrs != nil {
			return status.Append(sourceErrs, err)
		}
		sourceErrs = status.Append(sourceErrs, err)
		// At this point, `sourceErrs` only include the errors returned from `setSourceStatus`.
	}

	syncErrs := p.update(ctx, cos)
	if syncErrs != nil || !p.getCache().syncStatusUpdated {
		if err := p.setSyncStatus(ctx, state.commit, syncErrs); err != nil {
			glog.Errorf("Failed to set sync status: %v", err)
			syncErrs = status.Append(syncErrs, err)
			p.getCache().syncStatusUpdated = false
		} else {
			p.getCache().syncStatusUpdated = true
		}
	}

	// Only checkpoint our state *everything* succeeded, including status update.
	if sourceErrs == nil && syncErrs == nil {
		p.checkpoint(state.policyDir.OSPath())
		return nil
	}

	return status.Append(sourceErrs, syncErrs)
}
