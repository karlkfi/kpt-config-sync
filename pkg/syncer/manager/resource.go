/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package manager includes controller managers.
package manager

import (
	"context"
	"reflect"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"

	"github.com/google/nomos/pkg/util/discovery"

	"github.com/golang/glog"
	nomosapischeme "github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// RestartableManager is a controller manager that can be restarted based on the resources it syncs.
type RestartableManager interface {
	// UpdateSyncResources checks if the resources in Syncs have changed since last time we checked.
	// If they have, we stop the old manager and brings up a new one with controllers for sync-enabled resources.
	UpdateSyncResources(syncs []*v1.Sync, apirs *discovery.APIInfo, startErrCh chan error) error
	// Clear clears out the set of resource types ResourceManger is managing, without restarting the manager.
	Clear()
}

var _ RestartableManager = &GenericResourceManager{}

// GenericResourceManager is the Manager that is responsible for PolicyNode and ClusterPolicy controllers.
// Its controllers are responsible for reconciling declared state in sync-enabled resources with the
// actual state of K8S in the cluster.
type GenericResourceManager struct {
	manager.Manager
	// baseCfg is rest.Config that has no Sync-managed resources added to the scheme.
	baseCfg *rest.Config
	// syncEnabled are the resources that have sync enabled.
	syncEnabled map[schema.GroupVersionKind]bool
	// ctx is a cancelable ambient context used where necessary
	ctx context.Context
	// cancel is a cancelation function for ctx. May be nil if ctx is unavailable
	cancel context.CancelFunc
}

// NewGenericResourceManager returns a new GenericResourceManager for managing resources with sync enabled.
func NewGenericResourceManager(mgr manager.Manager, cfg *rest.Config) *GenericResourceManager {
	r := &GenericResourceManager{
		Manager:     mgr,
		baseCfg:     cfg,
		syncEnabled: make(map[schema.GroupVersionKind]bool),
	}
	r.initCtx()
	return r
}

func (r *GenericResourceManager) initCtx() {
	if r.cancel != nil {
		r.cancel()
	}
	// There doesn't seem to be a good other context to pass in here.
	r.ctx, r.cancel = context.WithCancel(context.Background())
}

// UpdateSyncResources implements RestartableManager.
func (r *GenericResourceManager) UpdateSyncResources(syncs []*v1.Sync, apirs *discovery.APIInfo,
	startErrCh chan error) error {
	actual := apirs.GroupVersionKinds(syncs...)
	if reflect.DeepEqual(actual, r.syncEnabled) {
		// The set of sync-enabled resources hasn't changed. There is no need to restart.
		return nil
	}
	r.initCtx()
	glog.Info("Stopping GenericResourceManager")

	var err error
	r.Manager, err = manager.New(rest.CopyConfig(r.baseCfg), manager.Options{})
	if err != nil {
		return errors.Wrap(err, "could not start GenericResourceManager")
	}
	if err := r.register(apirs, syncs); err != nil {
		return errors.Wrap(err, "could not start GenericResourceManager")
	}
	return r.startControllers(r.ctx, apirs, syncs, startErrCh)
}

// Clear implements RestartableManager.
func (r *GenericResourceManager) Clear() {
	r.syncEnabled = make(map[schema.GroupVersionKind]bool)
}

// register updates the scheme with resources declared in Syncs.
// This is needed to generate informers/listers for resources that are sync enabled.
func (r *GenericResourceManager) register(apirs *discovery.APIInfo, syncs []*v1.Sync) error {
	scheme := r.GetScheme()
	nomosapischeme.AddToScheme(scheme)
	enabled := apirs.GroupVersionKinds(syncs...)
	r.syncEnabled = enabled
	for gvk := range r.syncEnabled {
		if !scheme.Recognizes(gvk) {
			scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
			gvkList := schema.GroupVersionKind{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    gvk.Kind + "List",
			}
			scheme.AddKnownTypeWithName(gvkList, &unstructured.UnstructuredList{})
			metav1.AddToGroupVersion(scheme, gvk.GroupVersion())
		}
	}
	return nil
}

// startControllers starts all the controllers watching sync-enabled resources.
func (r *GenericResourceManager) startControllers(ctx context.Context, apirs *discovery.APIInfo, syncs []*v1.Sync,
	startErrCh chan error) error {
	namespace, cluster, err := r.resourceScopes(apirs)
	if err != nil {
		return errors.Wrap(err, "could not get resource scope information from discovery API")
	}

	decoder := decode.NewGenericResourceDecoder(r.GetScheme())
	if err := controller.AddPolicyNode(ctx, r, decoder, namespace); err != nil {
		return errors.Wrap(err, "could not create PolicyNode controller")
	}
	if err := controller.AddClusterPolicy(ctx, r, decoder, cluster); err != nil {
		return errors.Wrap(err, "could not create ClusterPolicy controller")
	}

	go func() {
		// Propagate errors with starting genericResourceManager up to the meta controller, so we can restart
		// genericResourceManager.
		startErrCh <- r.Start(ctx.Done())
	}()

	glog.Info("Starting GenericResourceManager")
	return nil
}

// resourceScopes returns two slices representing the namespace and cluster scoped resource types with sync enabled.
func (r *GenericResourceManager) resourceScopes(apirs *discovery.APIInfo) (map[schema.GroupVersionKind]runtime.Object,
	map[schema.GroupVersionKind]runtime.Object, error) {
	rts, err := r.resourceTypes()
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
func (r *GenericResourceManager) resourceTypes() (map[schema.GroupVersionKind]runtime.Object, error) {
	knownGVKs := r.GetScheme().AllKnownTypes()
	m := make(map[schema.GroupVersionKind]runtime.Object)
	for gvk := range r.syncEnabled {
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
