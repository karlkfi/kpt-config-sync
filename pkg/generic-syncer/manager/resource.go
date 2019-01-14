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
	"reflect"

	"github.com/golang/glog"
	nomosapischeme "github.com/google/nomos/clientgen/apis/scheme"
	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/generic-syncer/controller"
	"github.com/google/nomos/pkg/generic-syncer/decode"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// RestartableManager is a controller manager that can be restarted based on the resources it syncs.
type RestartableManager interface {
	// UpdateSyncResources checks if the resources in Syncs have changed since last time we checked.
	// If they have, we stop the old manager and brings up a new one with controllers for sync-enabled resources.
	UpdateSyncResources(syncs []*nomosv1alpha1.Sync, startErrCh chan error) error
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
	// stopCh is closed when we want to stop the embedded manager and its controllers.
	stopCh chan struct{}
}

// NewGenericResourceManager returns a new GenericResourceManager for managing resources with sync enabled.
func NewGenericResourceManager(mgr manager.Manager, cfg *rest.Config) *GenericResourceManager {
	return &GenericResourceManager{
		Manager:     mgr,
		baseCfg:     cfg,
		stopCh:      make(chan struct{}),
		syncEnabled: make(map[schema.GroupVersionKind]bool),
	}
}

// UpdateSyncResources implements RestartableManager.
func (r *GenericResourceManager) UpdateSyncResources(syncs []*nomosv1alpha1.Sync, startErrCh chan error) error {
	actual := GroupVersionKinds(syncs...)
	if reflect.DeepEqual(actual, r.syncEnabled) {
		// The set of sync-enabled resources hasn't changed. There is no need to restart.
		return nil
	}
	close(r.stopCh)
	r.stopCh = make(chan struct{})
	glog.Info("Stopping GenericResourceManager")

	var err error
	r.Manager, err = manager.New(rest.CopyConfig(r.baseCfg), manager.Options{})
	if err != nil {
		return errors.Wrap(err, "could not start GenericResourceManager")
	}
	r.register(syncs)
	return r.startControllers(syncs, startErrCh)
}

// Clear implements RestartableManager.
func (r *GenericResourceManager) Clear() {
	r.syncEnabled = make(map[schema.GroupVersionKind]bool)
}

// register updates the scheme with resources declared in Syncs.
// This is needed to generate informers/listers for resources that are sync enabled.
func (r *GenericResourceManager) register(syncs []*nomosv1alpha1.Sync) {
	scheme := r.GetScheme()
	nomosapischeme.AddToScheme(scheme)

	r.syncEnabled = GroupVersionKinds(syncs...)
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
}

// startControllers starts all the controllers watching sync-enabled resources.
func (r *GenericResourceManager) startControllers(syncs []*nomosv1alpha1.Sync, startErrCh chan error) error {
	namespace, cluster, err := r.resourceScopes()
	if err != nil {
		return errors.Wrap(err, "could not get resource scope information from discovery API")
	}

	decoder := decode.NewGenericResourceDecoder(r.GetScheme())
	if err := controller.AddPolicyNode(r, decoder, namespace); err != nil {
		return errors.Wrap(err, "could not create PolicyNode controller")
	}
	if err := controller.AddClusterPolicy(r, decoder, cluster); err != nil {
		return errors.Wrap(err, "could not create ClusterPolicy controller")
	}

	go func() {
		// Propagate errors with starting genericResourceManager up to the meta controller, so we can restart
		// genericResourceManager.
		startErrCh <- r.Start(r.stopCh)
	}()

	glog.Info("Starting GenericResourceManager")
	return nil
}

// resourceScopes returns two slices representing the namespace and cluster scoped resource types with sync enabled.
func (r *GenericResourceManager) resourceScopes() (map[schema.GroupVersionKind]runtime.Object,
	map[schema.GroupVersionKind]runtime.Object, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(r.GetConfig())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create discoveryclient")
	}
	groups, err := dc.ServerResources()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get api groups")
	}
	namespaceScoped := make(map[schema.GroupVersionKind]bool)
	for _, g := range groups {
		gv, grpErr := schema.ParseGroupVersion(g.GroupVersion)
		if grpErr != nil {
			// This shouldn't happen because we get these values from the server.
			return nil, nil, errors.Wrap(grpErr, "received invalid GroupVersion from server")
		}
		for _, apir := range g.APIResources {
			if apir.Namespaced {
				namespaceScoped[gv.WithKind(apir.Kind)] = true
			}
		}
	}

	rts, err := r.resourceTypes()
	if err != nil {
		return nil, nil, err
	}
	namespace := make(map[schema.GroupVersionKind]runtime.Object)
	cluster := make(map[schema.GroupVersionKind]runtime.Object)
	for gvk, obj := range rts {
		if namespaceScoped[gvk] {
			namespace[gvk] = obj
		} else {
			cluster[gvk] = obj
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
