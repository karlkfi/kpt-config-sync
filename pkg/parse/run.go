package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	webhookconfiguration "github.com/google/nomos/pkg/webhook/configuration"
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
			// The cached gitState will not be reset to avoid reading all the git files unnecessarily.
			state.resetAllButGitState()
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
				// The cached gitState will not be reset to avoid reading all the git files unnecessarily.
				state.resetAllButGitState()
				trigger = triggerManagementConflict
			} else if state.cache.needToRetry && state.cache.readyToRetry() {
				glog.Infof("The last reconciliation failed")
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
	oldPolicyDir := state.cache.git.policyDir
	// `read` is called no matter what the trigger is.
	if errs := read(ctx, p, trigger, state); errs != nil {
		state.invalidate(errs)
		return
	}

	newPolicyDir := state.cache.git.policyDir
	// The parse-apply-watch sequence will be skipped if the trigger type is `triggerReimport` and
	// there is no new git changes. The reasons are:
	//   * If a former parse-apply-watch sequence for policyDir succeeded, there is no need to run the sequence again;
	//   * If all the former parse-apply-watch sequences for policyDir failed, the next retry will call the sequence;
	//   * The retry logic tracks the number of reconciliation attempts failed with the same errors, and when
	//     the next retry should happen. Calling the parse-apply-watch sequence here makes the retry logic meaningless.
	if trigger == triggerReimport && oldPolicyDir == newPolicyDir {
		return
	}

	errs := parseAndUpdate(ctx, p, trigger, state)
	if errs != nil {
		state.invalidate(errs)
		return
	}

	// Only checkpoint the state after *everything* succeeded, including status update.
	state.checkpoint()
}

func read(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	commit, sourceErrs := readFromSource(ctx, p, trigger, state)
	if sourceErrs == nil {
		return nil
	}

	gs := gitStatus{
		commit: commit,
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
// readFromSource returns the git commit hash, and any encountered error.
func readFromSource(ctx context.Context, p Parser, trigger string, state *reconcilerState) (string, status.Error) {
	opts := p.options()
	start := time.Now()
	gs := gitState{}
	var sourceErrs status.Error
	gs.commit, gs.policyDir, sourceErrs = hydrate.SourceCommitAndDir(opts.GitDir, opts.PolicyDir, opts.reconcilerName)
	if sourceErrs != nil {
		return gs.commit, sourceErrs
	}

	if gs.policyDir == state.cache.git.policyDir {
		return gs.commit, nil
	}

	glog.Infof("New git changes (%s) detected, reset the cache", gs.policyDir.OSPath())

	// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
	state.resetCache()

	// Read all the files under state.policyDir
	sourceErrs = opts.readGitFiles(&gs)
	if sourceErrs == nil {
		// Set `state.cache.git` after `readGitFiles` succeeded
		state.cache.git = gs
	}
	metrics.RecordParserDuration(ctx, trigger, "read", metrics.StatusTagKey(sourceErrs), start)
	return gs.commit, sourceErrs
}

func parseSource(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	if state.cache.hasParserResult {
		return nil
	}

	start := time.Now()
	objs, sourceErrs := p.parseSource(ctx, state.cache.git)
	metrics.RecordParserDuration(ctx, trigger, "parse", metrics.StatusTagKey(sourceErrs), start)
	if sourceErrs != nil {
		return sourceErrs
	}

	state.cache.setParserResult(objs)

	err := webhookconfiguration.Update(ctx, p.options().k8sClient(), p.options().discoveryClient(), objs)
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

func parseAndUpdate(ctx context.Context, p Parser, trigger string, state *reconcilerState) status.MultiError {
	sourceErrs := parseSource(ctx, p, trigger, state)
	if sourceErrs != nil {
		newSourceStatus := gitStatus{
			commit: state.cache.git.commit,
			errs:   sourceErrs,
		}
		if err := p.setSourceStatus(ctx, state.sourceStatus, newSourceStatus); err != nil {
			sourceErrs = status.Append(sourceErrs, err)
		} else {
			state.sourceStatus = newSourceStatus
		}
		return sourceErrs
	}

	// If `parseSource` succeeded, we will not call `setSourceStatus` immediately.
	// Instead, we will call `p.options().update` first, and then update both the `Status.Source` and `Status.Sync` fields together in a single request.
	// Updating the `Status.Source` and `Status.Sync` fields in two requests may cause the second one to fail if these two requests are too close to each other.

	start := time.Now()
	syncErrs := p.options().update(ctx, &state.cache)
	metrics.RecordParserDuration(ctx, trigger, "update", metrics.StatusTagKey(syncErrs), start)

	newSourceStatus := gitStatus{
		commit: state.cache.git.commit,
		errs:   nil,
	}
	newSyncStatus := gitStatus{
		commit: state.cache.git.commit,
		errs:   syncErrs,
	}
	if err := p.setSourceAndSyncStatus(ctx, state.sourceStatus, newSourceStatus, state.syncStatus, newSyncStatus); err != nil {
		syncErrs = status.Append(syncErrs, err)
	} else {
		state.sourceStatus = newSourceStatus
		state.syncStatus = newSyncStatus
	}

	return syncErrs
}
