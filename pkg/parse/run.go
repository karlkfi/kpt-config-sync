package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	webhook2 "github.com/google/nomos/pkg/webhook"
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
	opts := p.options()
	tickerPoll := time.NewTicker(opts.pollingFrequency)
	tickerResync := time.NewTicker(opts.resyncPeriod)
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
			opts.resetCache()
			readAndParse(ctx, p, triggerResync)

		// it is time to re-import the configuration from the filesystem
		case <-tickerPoll.C:
			readAndParse(ctx, p, triggerReimport)

		// it is time to check whether the last parse-apply-watch loop failed or any watches need to be updated
		case <-tickerRetryOrWatchUpdate.C:
			var trigger string
			if opts.managementConflict() {
				glog.Infof("One of the watchers noticed a management conflict")
				// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
				opts.resetCache()
				trigger = triggerManagementConflict
			} else if opts.cache.needToRetry {
				glog.Infof("The last parse-apply-watch loop failed")
				trigger = triggerRetry
			} else if opts.needToUpdateWatch() {
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
	opts := p.options()
	start := time.Now()
	errs := read(ctx, p)
	metrics.RecordParserDuration(ctx, trigger, "read", metrics.StatusTagKey(errs), start)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		opts.invalidate(errs)
		return
	}

	start = time.Now()
	errs = parse(ctx, p)
	metrics.RecordParserDuration(ctx, trigger, "parse", metrics.StatusTagKey(errs), start)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		opts.invalidate(errs)
	}
}

// read reads the git commit and policyDir from the git repo, checks whether the gitstate in
// the cache is up-to-date. If the cache is not up-to-date, reads all the git files from the
// git repo and caches the gitstate.
//
// read does not log any error encountered, instead returns them to its caller.
func read(ctx context.Context, p Parser) status.MultiError {
	opts := p.options()
	state, err := opts.readGitCommitAndPolicyDir(opts.reconcilerName)
	if err != nil {
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			return status.Append(err, err2)
		}
		return err
	}

	if state.policyDir == opts.cache.git.policyDir {
		return nil
	}

	glog.Infof("New git changes (%s) detected, reset the cache", state.policyDir.OSPath())

	// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
	opts.resetCache()

	// Read all the files under state.policyDir
	if err := opts.readGitFiles(&state); err != nil {
		if err2 := p.setSourceStatus(ctx, state, err); err2 != nil {
			return status.Append(err, err2)
		}
		return err
	}

	// Set opts.cache.git after p.readGitFiles succeeded
	opts.cache.git = state
	return nil
}

// parse implements the parse-apply-watch loop.
//
// parse does not log any error encountered, instead returns them to its caller.
func parse(ctx context.Context, p Parser) status.MultiError {
	opts := p.options()

	// Parse the declared resources
	var cos []core.Object
	var sourceErrs, syncErrs status.MultiError
	var setSourceStatusErr, setSyncStatusErr error

	gitState := opts.cache.git

	if opts.cache.hasParserResult {
		cos = opts.cache.parserResult
	} else {
		cos, sourceErrs = p.parseSource(ctx, gitState)
		if sourceErrs == nil {
			opts.cache.setParserResult(cos)
		}
		err := webhook2.UpdateAdmissionWebhookConfiguration(ctx, p.options().k8sClient(), p.options().discoveryClient(), cos)
		if err != nil {
			// Don't block if updating the admission webhook fails.
			// Return an error instead if we remove the remediator as otherwise we
			// will simply never correct the type.
			// This should be treated as a warning (go/nomos-warning) once we have
			// that capability.
			glog.Errorf("Failed to update admission webhook: %v", err)
			// TODO(b/178605725): Handle case where multiple reconciler Pods try to
			//  create or update the Configuration simultaneously.
		}
	}

	if !opts.cache.sourceStatusUpdated {
		setSourceStatusErr = p.setSourceStatus(ctx, gitState, sourceErrs)
		if setSourceStatusErr == nil {
			opts.cache.sourceStatusUpdated = true
		}
		if sourceErrs != nil {
			return status.Append(sourceErrs, setSourceStatusErr)
		}
	}

	syncErrs = opts.update(ctx, cos)
	if !opts.cache.syncStatusUpdated {
		setSyncStatusErr = p.setSyncStatus(ctx, gitState.commit, syncErrs)
		if setSyncStatusErr == nil && syncErrs == nil {
			opts.cache.syncStatusUpdated = true
		}
	}

	// Only checkpoint our state *everything* succeeded, including status update.
	if sourceErrs == nil && syncErrs == nil && setSourceStatusErr == nil && setSyncStatusErr == nil {
		opts.checkpoint(gitState.policyDir.OSPath())
		return nil
	}

	return status.Append(sourceErrs, syncErrs, setSourceStatusErr, setSyncStatusErr)
}
