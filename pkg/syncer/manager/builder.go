package manager

import (
	"context"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/decode"
	syncerscheme "github.com/google/nomos/pkg/syncer/scheme"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// ControllerBuilder builds controllers. It is managed by RestartableManager, which is managed by a higher-level controller.
type ControllerBuilder interface {
	// StartControllers starts the relevant controllers using the RestartableManager to manage them.
	StartControllers(ctx context.Context, mgr manager.Manager, gvks map[schema.GroupVersionKind]bool) error
}

// SyncAwareBuilder creates controllers for managing resources with sync enabled.
// More info on Syncs available at go/nomos-syncless
type SyncAwareBuilder struct {
	Scoper discovery.Scoper
}

var _ ControllerBuilder = &SyncAwareBuilder{}

// NewSyncAwareBuilder returns a new syncAwareBuilder.
func NewSyncAwareBuilder() *SyncAwareBuilder {
	return &SyncAwareBuilder{discovery.Scoper{}}
}

// updateScheme updates the scheme with resources declared in Syncs.
// This is needed to generate informers/listers for resources that are sync enabled.
func (r *SyncAwareBuilder) updateScheme(scheme *runtime.Scheme, gvks map[schema.GroupVersionKind]bool) error {
	if err := v1.AddToScheme(scheme); err != nil {
		return err
	}
	syncerscheme.AddToSchemeAsUnstructured(scheme, gvks)
	return nil
}

// StartControllers starts all the controllers watching sync-enabled resources.
func (r *SyncAwareBuilder) StartControllers(ctx context.Context, mgr manager.Manager, gvks map[schema.GroupVersionKind]bool) error {
	if err := r.updateScheme(mgr.GetScheme(), gvks); err != nil {
		return errors.Wrap(err, "could not update the scheme")
	}

	namespace, cluster, err := syncerscheme.ResourceScopes(gvks, mgr.GetScheme(), r.Scoper)
	if err != nil {
		return errors.Wrap(err, "could not get resource scope information from discovery API")
	}

	decoder := decode.NewGenericResourceDecoder(mgr.GetScheme())
	if err := controller.AddNamespaceConfig(ctx, mgr, decoder, namespace); err != nil {
		return errors.Wrap(err, "could not create NamespaceConfig controller")
	}
	if err := controller.AddClusterConfig(ctx, mgr, decoder, cluster); err != nil {
		return errors.Wrap(err, "could not create ClusterConfig controller")
	}

	return nil
}
