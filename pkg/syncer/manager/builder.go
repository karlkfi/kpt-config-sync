package manager

import (
	"context"
	"reflect"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ControllerBuilder builds controllers. It is managed by SubManager, which is managed by a higher-level controller.
type ControllerBuilder interface {
	UpdateScheme(scheme *runtime.Scheme, gvks map[schema.GroupVersionKind]bool) error
	StartControllers(ctx context.Context, sm *SubManager, gvks map[schema.GroupVersionKind]bool, apirs *discovery.APIInfo) error
}

var _ ControllerBuilder = &SyncAwareBuilder{}

// SyncAwareBuilder creates controllers for managing resources with sync enabled.
type SyncAwareBuilder struct {
}

// NewSyncAwareBuilder returns a new SyncAwareBuilder.
func NewSyncAwareBuilder() *SyncAwareBuilder {
	return &SyncAwareBuilder{}
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
	namespace, cluster, err := r.resourceScopes(gvks, sm.GetScheme(), apirs)
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

	go func() {
		// Propagate errors with starting the SubManager up to the parent controller, so we can restart
		// SubManager.
		sm.errCh <- sm.Start(ctx.Done())
	}()

	glog.Info("Starting SyncAwareBuilder")
	return nil
}

// resourceScopes returns two slices representing the namespace and cluster scoped resource types with sync enabled.
func (r *SyncAwareBuilder) resourceScopes(gvks map[schema.GroupVersionKind]bool, scheme *runtime.Scheme,
	apirs *discovery.APIInfo) (map[schema.GroupVersionKind]runtime.Object, map[schema.GroupVersionKind]runtime.Object, error) {
	rts, err := r.resourceTypes(gvks, scheme)
	if err != nil {
		return nil, nil, err
	}
	namespace := make(map[schema.GroupVersionKind]runtime.Object)
	cluster := make(map[schema.GroupVersionKind]runtime.Object)
	for gvk, obj := range rts {
		switch apirs.GetScope(gvk) {
		case discovery.NamespaceScope:
			namespace[gvk] = obj
		case discovery.ClusterScope:
			cluster[gvk] = obj
		case discovery.UnknownScope:
			return nil, nil, errors.Errorf("Could not determine resource scope for %s", gvk)
		}
	}
	return namespace, cluster, nil
}

// resourceTypes returns all the sync enabled resources and the corresponding type stored in the scheme.
func (r *SyncAwareBuilder) resourceTypes(gvks map[schema.GroupVersionKind]bool,
	scheme *runtime.Scheme) (map[schema.GroupVersionKind]runtime.Object, error) {
	knownGVKs := scheme.AllKnownTypes()
	m := make(map[schema.GroupVersionKind]runtime.Object)
	for gvk := range gvks {
		rt, ok := knownGVKs[gvk]
		if !ok {
			return nil, errors.Errorf("trying to sync %q, which hasn't been registered in the scheme", gvk)
		}

		// If it's a resource with an unknown type at compile time, we need to specifically set the GroupVersionKind for it
		// when enabling the watch.
		if rt.AssignableTo(reflect.TypeOf(unstructured.Unstructured{})) {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(gvk)
			m[gvk] = u
		} else {
			m[gvk] = reflect.New(rt).Interface().(runtime.Object)
		}
	}
	return m, nil
}
