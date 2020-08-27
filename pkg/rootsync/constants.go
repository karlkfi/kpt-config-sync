package rootsync

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Name is the required name of any RootSync CR.
const Name = "root-sync"

// ObjectKey returns a key appropriate for fetching a RootSync.
// namespace.
func ObjectKey() client.ObjectKey {
	return client.ObjectKey{
		Namespace: v1.NSConfigManagementSystem,
		Name:      Name,
	}
}
