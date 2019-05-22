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
	Restart(string)
}

var _ RestartSignal = &RestartChannel{}

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
func (r *RestartChannel) Restart(source string) {
	// Send an event that forces the subManager to restart.
	// We have to shoehorn the source causing the restart into an event that the controller-runtime library understands. So,
	// we put the source in the Namespace field as a convention and know to only look at the namespace when it's an event that
	// was triggered by this method.
	r.channel <- event.GenericEvent{Meta: &metav1.ObjectMeta{Name: ForceRestart, Namespace: source}}
}

// Channel returns the internal channel.
func (r *RestartChannel) Channel() chan event.GenericEvent {
	return r.channel
}
