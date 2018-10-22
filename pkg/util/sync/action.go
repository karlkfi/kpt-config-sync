package sync

import (
	"github.com/google/go-cmp/cmp"
	typedv1alpha1 "github.com/google/nomos/clientgen/apis/typed/policyhierarchy/v1alpha1"
	listersv1alpha1 "github.com/google/nomos/clientgen/listers/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for Sync objects.
func NewActionSpec(client typedv1alpha1.NomosV1alpha1Interface, lister listersv1alpha1.SyncLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1alpha1.Sync),
		v1alpha1.SchemeGroupVersion,
		syncsEqual,
		client,
		lister)
}

func syncsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1alpha1.Sync)
	r := rhs.(*v1alpha1.Sync)
	return cmp.Equal(l.Spec, r.Spec)
}
