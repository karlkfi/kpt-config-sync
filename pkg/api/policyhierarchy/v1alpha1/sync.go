package v1alpha1

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewSync creates a sync object for consumption by the syncer, this will only populate the
// group and kind as those are the only fields the syncer presently consumes.
func NewSync(group, kind string) *Sync {
	var name string
	if group == "" {
		name = strings.ToLower(kind)
	} else {
		name = fmt.Sprintf("%s.%s", strings.ToLower(kind), group)
	}
	return &Sync{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       "Sync",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: SyncSpec{
			Group: group,
			Kind:  kind,
		},
	}
}

// GroupKind returns the schema.GroupKind object associated with the sync.
func (s *SyncSpec) GroupKind() schema.GroupKind {
	return schema.GroupKind{Group: s.Group, Kind: s.Kind}
}
