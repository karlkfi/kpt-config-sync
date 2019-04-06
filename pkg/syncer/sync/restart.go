package sync

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// ForceRestart is an invalid resource name used to signal that during Reconcile,
// the Sync Controller must restart the Sub Manager. Ensuring that the resource name
// is invalid ensures that we don't accidentally reconcile a resource that causes us
// to forcefully restart the SubManager.
const ForceRestart = "@restart"

// RestartSignal is a handle that causes the Sync controller to reconcile Syncs and
// forcefully restart the SubManager.
type RestartSignal interface {
	Restart()
}

// RestartChannel implements RestartSignal using a Channel.
type RestartChannel struct {
	channel chan event.GenericEvent
}

// NewRestartChannel returns a new RestartChannel.
func NewRestartChannel(channel chan event.GenericEvent) *RestartChannel {
	return &RestartChannel{
		channel: channel,
	}
}

// Restart implements RestartSignal
func (r *RestartChannel) Restart() {
	// Send an event that forces the subManager to restart.
	r.channel <- event.GenericEvent{Meta: &metav1.ObjectMeta{Name: ForceRestart}}
}

// Channel returns the internal channel.
func (r *RestartChannel) Channel() chan event.GenericEvent {
	return r.channel
}
