package manager

import (
	"context"
	"reflect"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ControllerBuilder builds controllers. It is managed by SubManager, which is managed by a higher-level controller.
type ControllerBuilder interface {
	// NeedsRestart returns true if the GroupVersionKinds do not match what the ControllerBuilder's controllers are
	// currently watching.
	NeedsRestart(toWatch map[schema.GroupVersionKind]bool) bool
	// UpdateScheme updates the scheme with the GroupVersionKinds.
	UpdateScheme(scheme *runtime.Scheme, toWatch map[schema.GroupVersionKind]bool) error
	// StartControllers starts the relevant controllers using the SubManager to manage them.
	StartControllers(ctx context.Context, sm *SubManager, gvks map[schema.GroupVersionKind]bool, apirs *discovery.APIInfo) error
}

var _ ControllerBuilder = &SyncAwareBuilder{}

// SyncAwareBuilder creates controllers for managing resources with sync enabled.
type SyncAwareBuilder struct {
	watching map[schema.GroupVersionKind]bool
}

// NewSyncAwareBuilder returns a new SyncAwareBuilder.
func NewSyncAwareBuilder() *SyncAwareBuilder {
	return &SyncAwareBuilder{}
}

// NeedsRestart implements ControllerBuilder.
func (r *SyncAwareBuilder) NeedsRestart(toWatch map[schema.GroupVersionKind]bool) bool {
	return !reflect.DeepEqual(r.watching, toWatch)
}

// UpdateScheme updates the scheme with resources declared in Syncs.
// This is needed to generate informers/listers for resources that are sync enabled.
func (r *SyncAwareBuilder) UpdateScheme(scheme *runtime.Scheme, gvks map[schema.GroupVersionKind]bool) error {
	if err := v1.AddToScheme(scheme); err != nil {
		return err
	}
	addToSchemeUnstructured(scheme, gvks)
	return nil
}

// StartControllers starts all the controllers watching sync-enabled resources.
func (r *SyncAwareBuilder) StartControllers(ctx context.Context, sm *SubManager,
	gvks map[schema.GroupVersionKind]bool, apirs *discovery.APIInfo) error {
	namespace, cluster, err := resourceScopes(gvks, sm.GetScheme(), apirs)
	if err != nil {
		return errors.Wrap(err, "could not get resource scope information from discovery API")
	}

	decoder := decode.NewGenericResourceDecoder(sm.GetScheme())
	if err := controller.AddNamespaceConfig(ctx, sm, decoder, namespace); err != nil {
		return errors.Wrap(err, "could not create NamespaceConfig controller")
	}
	if err := controller.AddClusterConfig(ctx, sm, decoder, cluster); err != nil {
		return errors.Wrap(err, "could not create ClusterConfig controller")
	}

	r.watching = gvks
	return nil
}
