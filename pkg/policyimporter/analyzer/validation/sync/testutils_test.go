package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
)

// ToFileSync converts a FileGroupVersionKindHierarchySync into a FileSync.
// Unsuitable implementation for production purposes.
func toFileSync(sync FileGroupVersionKindHierarchySync) FileSync {
	version := v1alpha1.SyncVersion{Version: sync.GroupVersionKind().Version}

	kind := v1alpha1.SyncKind{
		Kind:          sync.GroupVersionKind().Kind,
		HierarchyMode: sync.HierarchyMode,
		Versions:      []v1alpha1.SyncVersion{version},
	}

	group := v1alpha1.SyncGroup{
		Group: sync.GroupVersionKind().Group,
		Kinds: []v1alpha1.SyncKind{kind},
	}

	return FileSync{
		source: sync.RelativeSlashPath(),
		Sync: &v1alpha1.Sync{
			Spec: v1alpha1.SyncSpec{
				Groups: []v1alpha1.SyncGroup{group},
			},
		},
	}
}
