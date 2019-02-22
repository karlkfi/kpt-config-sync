package sync

import (
	"github.com/google/go-cmp/cmp"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for Sync objects.
func NewActionSpec(client typedv1.NomosV1Interface, lister listersv1.SyncLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1.Sync),
		v1.SchemeGroupVersion,
		syncsEqual,
		client,
		lister)
}

func syncsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.Sync)
	r := rhs.(*v1.Sync)
	return cmp.Equal(l.Spec, r.Spec)
}
