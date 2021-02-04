package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/webhook"
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
	state := &reconcilerState{}
	for {
		select {
		case <-ctx.Done():
			return

		// it is time to reapply the configuration even if no changes have been detected
		// This case should be checked first since it resets the cache
		case <-tickerResync.C:
			glog.Infof("It is time for a force-resync")
			// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
			state.resetCache()
			run(ctx, p, triggerResync, state)

		// it is time to re-import the configuration from the filesystem
		case <-tickerPoll.C:
			run(ctx, p, triggerReimport, state)

		// it is time to check whether the last parse-apply-watch loop failed or any watches need to be updated
		case <-tickerRetryOrWatchUpdate.C:
			var trigger string
			if opts.managementConflict() {
				glog.Infof("One of the watchers noticed a management conflict")
				// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
				state.resetCache()
				trigger = triggerManagementConflict
			} else if state.cache.needToRetry {
				glog.Infof("The last parse-apply-watch loop failed")
				trigger = triggerRetry
			} else if opts.needToUpdateWatch() {
				glog.Infof("Some watches need to be updated")
				trigger = triggerWatchUpdate
			} else {
				continue
			}
			run(ctx, p, trigger, state)
		}
	}
}

func run(ctx context.Context, p Parser, trigger string, state *reconcilerState) {
	start := time.Now()
	sourceErrs := read(ctx, p, state)
	metrics.RecordParserDuration(ctx, trigger, "read", metrics.StatusTagKey(sourceErrs), start)
	if sourceErrs != nil {
		gs := gitStatus{
			commit: "",
			errs:   sourceErrs,
		}
		// Only call `updateSourceStatus` if `read` fails.
		// If `read` succeeds, `parse` may still fail.
		setSourceStatusErr := p.setSourceStatus(ctx, state.sourceStatus, gs)
		if setSourceStatusErr == nil {
			state.sourceStatus = gs
		}

		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		state.invalidate(status.Append(sourceErrs, setSourceStatusErr))
		return
	}

	parseAndUpdate(ctx, p, trigger, state)
}

func parseAndUpdate(ctx context.Context, p Parser, trigger string, state *reconcilerState) {
	start := time.Now()
	sourceErrs := parse(ctx, p, state)
	metrics.RecordParserDuration(ctx, trigger, "parse", metrics.StatusTagKey(sourceErrs), start)

	gs := gitStatus{
		commit: state.cache.git.commit,
		errs:   sourceErrs,
	}
	// After `parse` is done, we have all the source errors, and call `updateSourceStatus` even if
	// sourceErrs is nil to make sure that the `Status.Source` field of RepoSync/RootSync is up-to-date.
	setSourceStatusErr := p.setSourceStatus(ctx, state.sourceStatus, gs)
	if setSourceStatusErr == nil {
		state.sourceStatus = gs
	}

	// Only returns if `sourceErrs` is not nil.
	// If `sourceErrs` is nil, but `setSourceStatusErr` is not, we will not return here to make sure
	// that `update` is called.
	if sourceErrs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		state.invalidate(status.Append(sourceErrs, setSourceStatusErr))
		return
	}

	start = time.Now()
	syncErrs := p.options().update(ctx, &state.cache)
	metrics.RecordParserDuration(ctx, trigger, "update", metrics.StatusTagKey(syncErrs), start)
	gs.errs = syncErrs
	setSyncStatusErr := p.setSyncStatus(ctx, state.syncStatus, gs)
	if setSyncStatusErr == nil {
		state.syncStatus = gs
	}

	errs := status.Append(syncErrs, setSourceStatusErr, setSyncStatusErr)
	if errs != nil {
		// Invalidate state on error since this could be the result of switching
		// branches or some other operation where inverting the operation would
		// result in repeating a previous state that was checkpointed.
		state.invalidate(errs)
		return
	}

	// Only checkpoint our state *everything* succeeded, including status update.
	state.checkpoint()
}

// read reads the git commit and policyDir from the git repo, checks whether the gitstate in
// the cache is up-to-date. If the cache is not up-to-date, reads all the git files from the
// git repo.
func read(ctx context.Context, p Parser, state *reconcilerState) status.Error {
	opts := p.options()
	gitState, sourceErrs := opts.readGitCommitAndPolicyDir(opts.reconcilerName)
	if sourceErrs != nil {
		return sourceErrs
	}

	if gitState.policyDir == state.cache.git.policyDir {
		return nil
	}

	glog.Infof("New git changes (%s) detected, reset the cache", gitState.policyDir.OSPath())

	// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
	state.resetCache()

	// Read all the files under state.policyDir
	sourceErrs = opts.readGitFiles(&gitState)
	if sourceErrs == nil {
		// Set `state.cache.git` after `readGitFiles` succeeded
		state.cache.git = gitState
	}
	return sourceErrs
}

func parse(ctx context.Context, p Parser, state *reconcilerState) status.MultiError {
	if state.cache.hasParserResult {
		return nil
	}

	cos, sourceErrs := p.parseSource(ctx, state.cache.git)
	if sourceErrs != nil {
		return sourceErrs
	}

	state.cache.setParserResult(cos)

	err := webhook.UpdateAdmissionWebhookConfiguration(ctx, p.options().k8sClient(), p.options().discoveryClient(), cos)
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
	return nil
}
