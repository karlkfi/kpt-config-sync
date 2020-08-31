package applier

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Run periodically syncs the resource state in the API server with the git resource in every
// ResyncPeriod until the StopChannel is called.
func (a *Applier) Run(ctx context.Context, resyncPeriod time.Duration, stopChannel <-chan struct{}) {
	ticker := time.NewTicker(resyncPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stopChannel:
			return
		case <-ticker.C:
		}
		errs := a.Refresh(ctx)
		if errs != nil {
			glog.Errorf("applier run failed: %v", errs)
		} else {
			glog.V(4).Infoln("applier run succeeded.")
		}
		now := time.Now()
		if a.isRootApplier() {
			a.setRootSyncErrs(ctx, errs, now)
		} else {
			a.setRepoSyncErrs(ctx, errs, now)
		}
		glog.V(2).Infof("applier run finished at %s", now.Format(time.RFC3339))
	}
}

func (a *Applier) setRepoSyncErrs(ctx context.Context, errs status.MultiError, now time.Time) {
	var rs v1alpha1.RepoSync
	if err := a.client.Get(ctx, reposync.ObjectKey(a.scope), &rs); err != nil {
		glog.Errorf("Failed to get RepoSync for %s applier refresh: %v", a.scope, err)
		return
	}

	rs.Status.Sync.LastUpdate = metav1.NewTime(now)
	rs.Status.Sync.Errors = status.ToCSE(errs)
	if err := a.client.Status().Update(ctx, &rs); err != nil {
		glog.Errorf("Failed to update RepoSync status from %s applier refresh: %v", a.scope, err)
	}
}

func (a *Applier) setRootSyncErrs(ctx context.Context, errs status.MultiError, now time.Time) {
	var rs v1alpha1.RootSync
	if err := a.client.Get(ctx, rootsync.ObjectKey(), &rs); err != nil {
		glog.Errorf("Failed to get RootSync for %s applier refresh: %v", a.scope, err)
		return
	}

	rs.Status.Sync.LastUpdate = metav1.NewTime(now)
	rs.Status.Sync.Errors = status.ToCSE(errs)
	if err := a.client.Status().Update(ctx, &rs); err != nil {
		glog.Errorf("Failed to update RootSync status from %s applier refresh: %v", a.scope, err)
	}
}
