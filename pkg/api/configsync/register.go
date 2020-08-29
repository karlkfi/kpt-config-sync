package configsync

const (

	// GroupName is the name of the group of configsync resources.
	GroupName = "configsync.gke.io"

	// ControllerNamespace is the Namespace used for Nomos controllers
	ControllerNamespace = "config-management-system"

	// RepoSyncKind is the string constant for the RepoSync GroupVersionKind
	RepoSyncKind = "RepoSync"
)

// IsControllerNamespace returns true if the namespace is the ACM Controller Namespace.
func IsControllerNamespace(name string) bool {
	// For now we only forbid syncing the Namespace containing the ACM controllers.
	return name == ControllerNamespace
}
