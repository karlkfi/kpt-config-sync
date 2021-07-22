package parse

import (
	"context"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
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
	var syncDir cmpath.Absolute
	gs := gitStatus{}
	gs.commit, syncDir, gs.errs = hydrate.SourceCommitAndDir(p.options().GitDir, p.options().PolicyDir, p.options().reconcilerName)

	// If failed to fetch the source commit and directory, set `.status.source` to fail early.
	// Otherwise, set `.status.rendering` before `.status.source` because the parser needs to
	// read and parse the configs after rendering is done and there might have errors.
	if gs.errs != nil {
		setSourceStatusErr := p.setSourceStatus(ctx, state.sourceStatus, gs)
		if setSourceStatusErr == nil {
			state.sourceStatus = gs
		}
		state.invalidate(status.Append(gs.errs, setSourceStatusErr))
		return
	}

	rs := renderingStatus{
		commit: gs.commit,
	}
	// set the rendering status by checking the done file.
	doneFilePath := p.options().RepoRoot.Join(cmpath.RelativeSlash(hydrate.DoneFile)).OSPath()
	if _, err := os.Stat(doneFilePath); err != nil {
		var readErrs status.MultiError
		if os.IsNotExist(err) {
			rs.phase = v1alpha1.RenderingInProgress
			readErrs = status.HydrationInProgress(gs.commit)
		} else {
			rs.phase = v1alpha1.RenderingFailed
			rs.errs = status.HydrationError.Wrap(err).
				Sprintf("unable to read the done file: %s", doneFilePath).
				Build()
			readErrs = rs.errs
		}
		setRenderingStatusErr := p.setRenderingStatus(ctx, state.renderingStatus, rs)
		if setRenderingStatusErr == nil {
			state.renderingStatus = rs
		}
		state.invalidate(status.Append(readErrs, setRenderingStatusErr))
		return
	}

	// rendering is done, starts to read the source or hydrated configs.
	oldPolicyDir := state.cache.git.policyDir
	// `read` is called no matter what the trigger is.
	sourceState := gitState{
		commit:    gs.commit,
		policyDir: syncDir,
	}
	if errs := read(ctx, p, trigger, state, sourceState); errs != nil {
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

// read reads config files from source if no rendering is needed, or from hydrated output if rendering is done.
// It also updates the .status.rendering and .status.source fields.
func read(ctx context.Context, p Parser, trigger string, state *reconcilerState, sourceState gitState) status.MultiError {
	hydrationStatus, sourceStatus := readFromSource(ctx, p, trigger, state, sourceState)
	// update the rendering status before source status because the parser needs to
	// read and parse the configs after rendering is done and there might have errors.
	setRenderingStatusErr := p.setRenderingStatus(ctx, state.renderingStatus, hydrationStatus)
	if setRenderingStatusErr == nil {
		state.renderingStatus = hydrationStatus
	}
	renderingErrs := status.Append(hydrationStatus.errs, setRenderingStatusErr)
	if renderingErrs != nil {
		return renderingErrs
	}

	if sourceStatus.errs == nil {
		return nil
	}

	// Only call `setSourceStatus` if `readFromSource` fails.
	// If `readFromSource` succeeds, `parse` may still fail.
	setSourceStatusErr := p.setSourceStatus(ctx, state.sourceStatus, sourceStatus)
	if setSourceStatusErr == nil {
		state.sourceStatus = sourceStatus
	}
	return status.Append(sourceStatus.errs, setSourceStatusErr)
}

// readFromSource reads the source or hydrated configs, checks whether the gitState in
// the cache is up-to-date. If the cache is not up-to-date, reads all the source or hydrated files.
// readFromSource returns the rendering status and source status.
func readFromSource(ctx context.Context, p Parser, trigger string, state *reconcilerState, sourceState gitState) (renderingStatus, gitStatus) {
	opts := p.options()
	start := time.Now()

	hydrationStatus := renderingStatus{
		commit: sourceState.commit,
	}
	sourceStatus := gitStatus{
		commit: sourceState.commit,
	}

	// Check if the hydratedRoot directory exists.
	// If exists, read the hydrated directory. Otherwise, read the source directory.
	absHydratedRoot, err := cmpath.AbsoluteOS(opts.HydratedRoot)
	if err != nil {
		hydrationStatus.phase = v1alpha1.RenderingFailed
		hydrationStatus.errs = status.HydrationError.Wrap(err).
			Sprint("hydrated-dir must be an absolute path").Build()
		return hydrationStatus, sourceStatus
	}

	var hydrationErr error
	if _, err := os.Stat(absHydratedRoot.OSPath()); err == nil {
		sourceState, hydrationErr = opts.readHydratedDir(absHydratedRoot, opts.HydratedLink, opts.reconcilerName)
		if hydrationErr != nil {
			hydrationStatus.phase = v1alpha1.RenderingFailed
			hydrationStatus.errs = status.HydrationError.Wrap(hydrationErr).Build()
			return hydrationStatus, sourceStatus
		}
		hydrationStatus.phase = v1alpha1.RenderingSucceeded
	} else if !os.IsNotExist(err) {
		hydrationStatus.phase = v1alpha1.RenderingFailed
		hydrationStatus.errs = status.HydrationError.Wrap(err).
			Sprintf("unable to evaluate the hydrated path %s", absHydratedRoot.OSPath()).Build()
		return hydrationStatus, sourceStatus
	} else {
		hydrationStatus.phase = v1alpha1.RenderingSkipped
	}

	if sourceState.policyDir == state.cache.git.policyDir {
		return hydrationStatus, sourceStatus
	}

	glog.Infof("New git changes (%s) detected, reset the cache", sourceState.policyDir.OSPath())

	// Reset the cache to make sure all the steps of a parse-apply-watch loop will run.
	state.resetCache()

	// Read all the files under state.policyDir
	sourceStatus.errs = opts.readConfigFiles(&sourceState, hydrationStatus.phase)
	if sourceStatus.errs == nil {
		// Set `state.cache.git` after `readConfigFiles` succeeded
		state.cache.git = sourceState
	}
	metrics.RecordParserDuration(ctx, trigger, "read", metrics.StatusTagKey(sourceStatus.errs), start)
	return hydrationStatus, sourceStatus
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
