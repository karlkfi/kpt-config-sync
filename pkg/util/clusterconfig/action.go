package clusterconfig

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	typedv1 "github.com/google/nomos/clientgen/apis/typed/configmanagement/v1"
	listersv1 "github.com/google/nomos/clientgen/listers/configmanagement/v1"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewActionSpec returns a ReflectiveActionSpec appropriate for ClusterConfig objects.
func NewActionSpec(client typedv1.ConfigmanagementV1Interface, lister listersv1.ClusterConfigLister) *action.ReflectiveActionSpec {
	return action.NewSpec(
		new(v1.ClusterConfig),
		v1.SchemeGroupVersion,
		clusterConfigsEqual,
		client,
		lister)
}

var cpsIgnore = []cmp.Option{
	cmpopts.IgnoreFields(v1.ClusterConfigSpec{}, "Token", "ImportTime"),
}

func clusterConfigsEqual(lhs runtime.Object, rhs runtime.Object) bool {
	l := lhs.(*v1.ClusterConfig)
	r := rhs.(*v1.ClusterConfig)
	return cmp.Equal(l.Spec, r.Spec, cpsIgnore...)
}
