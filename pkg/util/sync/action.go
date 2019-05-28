package sync

import (
	"github.com/google/go-cmp/cmp"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for Sync objects.
func NewActionSpec(client typedv1.ConfigmanagementV1Interface, lister listersv1.SyncLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1.Sync),
		v1.SchemeGroupVersion,
		SyncsEqual,
		client,
		lister)
}

// SyncsEqual returns true if the syncs are equivalent.
func SyncsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.Sync)
	r := rhs.(*v1.Sync)
	return cmp.Equal(l.Spec, r.Spec) && action.ObjectMetaEqual(l, r)
}
