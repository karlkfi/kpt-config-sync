package sync

import (
	"context"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/decode"
	syncerscheme "github.com/google/nomos/pkg/syncer/scheme"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/watch"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// syncAwareBuilder creates controllers for managing resources with sync enabled.
// More info on Syncs available at go/nomos-syncless
type syncAwareBuilder struct {
	scoper discovery.Scoper
}

var _ watch.ControllerBuilder = &syncAwareBuilder{}

// newSyncAwareBuilder returns a new syncAwareBuilder.
func newSyncAwareBuilder() *syncAwareBuilder {
	return &syncAwareBuilder{discovery.Scoper{}}
}

// updateScheme updates the scheme with resources declared in Syncs.
// This is needed to generate informers/listers for resources that are sync enabled.
func (r *syncAwareBuilder) updateScheme(scheme *runtime.Scheme, gvks map[schema.GroupVersionKind]bool) error {
	if err := v1.AddToScheme(scheme); err != nil {
		return err
	}
	syncerscheme.AddToSchemeAsUnstructured(scheme, gvks)
	return nil
}

// StartControllers starts all the controllers watching sync-enabled resources.
func (r *syncAwareBuilder) StartControllers(ctx context.Context, mgr manager.Manager, gvks map[schema.GroupVersionKind]bool, mgrInitTime metav1.Time) error {
	if err := r.updateScheme(mgr.GetScheme(), gvks); err != nil {
		return errors.Wrap(err, "could not update the scheme")
	}

	namespace, cluster, err := syncerscheme.ResourceScopes(gvks, mgr.GetScheme(), r.scoper)
	if err != nil {
		return errors.Wrap(err, "could not get resource scope information from discovery API")
	}

	decoder := decode.NewGenericResourceDecoder(mgr.GetScheme())
	if err := controller.AddNamespaceConfig(ctx, mgr, decoder, namespace, mgrInitTime); err != nil {
		return errors.Wrap(err, "could not create NamespaceConfig controller")
	}
	if err := controller.AddClusterConfig(ctx, mgr, decoder, cluster, mgrInitTime); err != nil {
		return errors.Wrap(err, "could not create ClusterConfig controller")
	}

	return nil
}
