package reconcile

import (
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/status"
)

// fightLogger is used to log errors about fights from fightDetector at most
// once every 60 seconds. It has similar performance characteristics as
// fightDetector.
//
// Instantiate with newFightLogger().
type fightLogger struct {
	// lastLogged is when fightLogger last logged about a given API resource.
	lastLogged map[gknn]time.Time
}

func newFightLogger() fightLogger {
	return fightLogger{
		lastLogged: make(map[gknn]time.Time),
	}
}

// markUpdated marks that API resource `resource` was updated at time `now`.
// If the estimated frequency of updates is greater than `fightThreshold`, logs
// this to glog.Warning. The log message appears at most once per minute.
//
// Returns true if the new estimated update frequency is at least `fightThreshold`.
func (d *fightLogger) logFight(now time.Time, err status.ResourceError) bool {
	resource := err.Resources()[0] // There is only ever one resource per fight error.
	i := gknn{
		gk:        resource.GroupVersionKind().GroupKind(),
		namespace: resource.GetNamespace(),
		name:      resource.GetName(),
	}

	if now.Sub(d.lastLogged[i]) <= time.Minute {
		return false
	}

	glog.Warning(err)
	d.lastLogged[i] = now
	return true
}
