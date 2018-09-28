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
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

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
		Manager: mgr,
		baseCfg: cfg,
		stopCh:  make(chan struct{}),
	}
}

// ResourcesChanged returns true if the resources in Syncs have changed since last time we checked.
// When this happens we need to stop the Manager and create a new one for the current resources with sync enabled.
func (r *GenericResourceManager) ResourcesChanged(syncs *nomosv1alpha1.SyncList) bool {
	actual := groupVersionKinds(syncs)
	return !reflect.DeepEqual(actual, r.syncEnabled)
}

// Restart stops the old manager and brings up a new one with controllers for sync-enabled resources.
// This is called whenever the resources being synced in a cluster has changed.
func (r *GenericResourceManager) Restart(syncs *nomosv1alpha1.SyncList, startErrCh chan error) error {
	close(r.stopCh)
	r.stopCh = make(chan struct{})
	glog.Info("Stopping GenericResourceManager")

	var err error
	r.Manager, err = manager.New(rest.CopyConfig(r.baseCfg), manager.Options{})
	if err != nil {
		return errors.Wrap(err, "could not start GenericResourceManager")
	}
	r.register(syncs)
	return r.startControllers(startErrCh)
}

// Clear clears out the set of resource types ResourceManger is managing.
func (r *GenericResourceManager) Clear() {
	r.syncEnabled = nil
}

// register updates the scheme with resources declared in Syncs.
// This is needed to generate informers/listers for resources that are sync enabled.
func (r *GenericResourceManager) register(syncs *nomosv1alpha1.SyncList) {
	scheme := r.GetScheme()
	nomosapischeme.AddToScheme(scheme)

	r.syncEnabled = groupVersionKinds(syncs)
	for gvk := range r.syncEnabled {
		if !scheme.Recognizes(gvk) {
			scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
			// TODO: see if we can avoid akwardly creating a list Kind.
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
func (r *GenericResourceManager) startControllers(startErrCh chan error) error {
	namespace, cluster, err := r.resourceScopes()
	if err != nil {
		return errors.Wrap(err, "could not get resource scope information from discovery API")
	}

	if err := controller.AddPolicyNode(r, namespace); err != nil {
		return errors.Wrap(err, "could not create PolicyNode controllers")
	}
	if err := controller.AddClusterPolicy(r, cluster); err != nil {
		return errors.Wrap(err, "could not create ClusterPolicy controllers")
	}

	go func() {
		// Propagate errors with starting genericResourceManager up to the meta controller, so we can restart
		// genericResourceManager.
		startErrCh <- r.Start(r.stopCh)
	}()

	glog.Info("Starting GenericResourceManager")
	return nil
}

// resourceScopes returns two slices representing the namespaced and cluster scoped resource types with sync enabled.
func (r *GenericResourceManager) resourceScopes() (namespace []schema.GroupVersionKind, cluster []schema.GroupVersionKind,
	err error) {
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

	for gvk := range r.syncEnabled {
		if namespaceScoped[gvk] {
			namespace = append(namespace, gvk)
		} else {
			cluster = append(cluster, gvk)
		}
	}
	return namespace, cluster, nil
}

// groupVersionKinds returns a set of GroupVersionKind represented by the SyncList.
func groupVersionKinds(syncs *nomosv1alpha1.SyncList) map[schema.GroupVersionKind]bool {
	gvks := make(map[schema.GroupVersionKind]bool)
	for _, sync := range syncs.Items {
		for _, g := range sync.Spec.Groups {
			k := g.Kinds
			for _, v := range k.Versions {
				gvk := schema.GroupVersionKind{
					Group:   g.Group,
					Version: v.Version,
					Kind:    k.Kind,
				}
				gvks[gvk] = true
			}
		}
	}
	return gvks
}
