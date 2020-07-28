package applier

import (
	"context"
	"time"

	"github.com/golang/glog"
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
		err := a.Refresh(ctx)
		if err != nil {
			glog.Errorf("applier run failed: %v", err)
		} else {
			glog.V(4).Infoln("applier run succeeded.")
		}
		now := time.Now()
		glog.V(2).Infof("applier run finished at %s", now.Format(time.RFC3339))
	}
}
