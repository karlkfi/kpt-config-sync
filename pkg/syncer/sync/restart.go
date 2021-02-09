package sync

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// ForceRestart is an invalid resource name used to signal that during Reconcile,
// the Sync Controller must restart the Sub Manager. Ensuring that the resource name
// is invalid ensures that we don't accidentally reconcile a resource that causes us
// to forcefully restart the SubManager.
const forceRestart = "@restart"

// RestartSignal is a handle that causes the Sync controller to reconcile Syncs and
// forcefully restart the SubManager.
type RestartSignal interface {
	Restart(string)
}

var _ RestartSignal = RestartChannel(nil)

// RestartChannel implements RestartSignal using a Channel.
type RestartChannel chan event.GenericEvent

// Restart implements RestartSignal
func (r RestartChannel) Restart(source string) {
	// Send an event that forces the subManager to restart.
	// We have to shoehorn the source causing the restart into an event that the controller-runtime library understands. So,
	// we put the source in the Namespace field as a convention and know to only look at the namespace when it's an event that
	// was triggered by this method.
	// TODO(b/179816931): Not an intended use case for GenericEvent. Refactor.
	u := &unstructured.Unstructured{}
	u.SetNamespace(source)
	u.SetName(forceRestart)
	r <- event.GenericEvent{Object: u}
}
