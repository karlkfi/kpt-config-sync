package fake

import (
	"github.com/google/nomos/pkg/util/watch"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RestartableManagerRecorder records whether each instance of Restart was
// forced.
type RestartableManagerRecorder struct {
	Restarts []bool
}

// Restart implements watch.RestartableManager.
func (r *RestartableManagerRecorder) Restart(_ map[schema.GroupVersionKind]bool, force bool) (bool, error) {
	r.Restarts = append(r.Restarts, force)
	return false, nil
}

var _ watch.RestartableManager = &RestartableManagerRecorder{}
