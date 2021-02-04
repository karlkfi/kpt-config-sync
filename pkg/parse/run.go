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
	errs := read(ctx, p, trigger, state)
	if errs != nil {
		state.invalidate(errs)
		return
	}

	errs = parse(ctx, p, trigger, state)
	// errs includes the error(s) returned from `parseSource` and `p.setSourceStatus`.
	// if `parseSource` succeeds, but `p.setSourceStatus` fails, the function calls `state.invalidate` and returns.
	// Moving forward to `update` in this case may result in confusing `Status` field in RepoSync/RootSync.
	// Imagine we have two git commits: A and B (which was committed after A). First, commit A is fully synced, and
	// is recorded in both the `Status.Source` and `Status.Sync` fields. Then while reconciling commit B, `parseSource`
	// succeeds, but `p.setSourceStatus` fails, which leaves `Status.Source.Commit` as A. If we move forward to
	// `update` here instead of returning, and both `update` and `p.setSyncStatus` succeed, `Status.Sync.Commit`
	// will be updated to B. Then the users will see that `Status.Sync.Commit` includes a commit committed after
	// the commit in `Status.Source.Commit`, and get confused.
	if errs != nil {
		state.invalidate(errs)
		return
	}

	errs = update(ctx, p, trigger, state)
	if errs != nil {
		state.invalidate(errs)
		return
	}

	// Only checkpoint the state after *everything* succeeded, including status update.
	state.checkpoint()
}

func read(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	sourceErrs := readFromSource(ctx, p, trigger, state)
	if sourceErrs == nil {
		return nil
	}

	gs := gitStatus{
		commit: "",
		errs:   sourceErrs,
	}
	// Only call `setSourceStatus` if `readFromSource` fails.
	// If `readFromSource` succeeds, `parse` may still fail.
	setSourceStatusErr := p.setSourceStatus(ctx, state.sourceStatus, gs)
	if setSourceStatusErr == nil {
		state.sourceStatus = gs
	}
	return status.Append(sourceErrs, setSourceStatusErr)
}

// readFromSource reads the git commit and policyDir from the git repo, checks whether the gitstate in
// the cache is up-to-date. If the cache is not up-to-date, reads all the git files from the
// git repo.
func readFromSource(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.Error {
	opts := p.options()
	start := time.Now()
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
	metrics.RecordParserDuration(ctx, trigger, "read", metrics.StatusTagKey(sourceErrs), start)
	return sourceErrs
}

func parse(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	sourceErrs := parseSource(ctx, p, trigger, state)
	gs := gitStatus{
		commit: state.cache.git.commit,
		errs:   sourceErrs,
	}
	// After `parse` is done, we have all the source errors, and call `setSourceStatus` even if
	// sourceErrs is nil to make sure that the `Status.Source` field of RepoSync/RootSync is up-to-date.
	setSourceStatusErr := p.setSourceStatus(ctx, state.sourceStatus, gs)
	if setSourceStatusErr == nil {
		state.sourceStatus = gs
	}
	return status.Append(sourceErrs, setSourceStatusErr)
}

func parseSource(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	if state.cache.hasParserResult {
		return nil
	}

	start := time.Now()
	cos, sourceErrs := p.parseSource(ctx, state.cache.git)
	metrics.RecordParserDuration(ctx, trigger, "parse", metrics.StatusTagKey(sourceErrs), start)
	if sourceErrs != nil {
		return sourceErrs
	}

	state.cache.setParserResult(cos)

	// TODO(b/179505403): Re-enable after 1.6.2 by replacing objs: nil.
	err := webhook.UpdateAdmissionWebhookConfiguration(ctx, p.options().k8sClient(), p.options().discoveryClient(), nil)
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

func update(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	start := time.Now()
	syncErrs := p.options().update(ctx, &state.cache)
	metrics.RecordParserDuration(ctx, trigger, "update", metrics.StatusTagKey(syncErrs), start)
	gs := gitStatus{
		commit: state.cache.git.commit,
		errs:   syncErrs,
	}
	setSyncStatusErr := p.setSyncStatus(ctx, state.syncStatus, gs)
	if setSyncStatusErr == nil {
		state.syncStatus = gs
	}

	return status.Append(syncErrs, setSyncStatusErr)
}
