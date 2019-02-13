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

// Package metasync includes controllers and reconcilers responsible for
// managing other controllers based on Syncs.
package metasync

import (
	"github.com/golang/glog"
	nomosapischeme "github.com/google/nomos/clientgen/apis/scheme"
	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var unaryHandler = &handler.EnqueueRequestsFromMapFunc{
	ToRequests: handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: apimachinerytypes.NamespacedName{Name: "item"}}}
	}),
}

// AddMetaController adds MetaController to the manager.
func AddMetaController(mgr manager.Manager, stopCh <-chan struct{}) error {
	// Set up Scheme for nomos resources.
	nomosapischeme.AddToScheme(mgr.GetScheme())

	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrapf(err, "failed to create discoveryclient")
	}
	// Set up a meta controller that restarts GenericResource controllers when Syncs change.
	startErrCh := make(chan error)
	clientFactory := func() (runtimeclient.Client, error) {
		cfg := mgr.GetConfig()
		mapper, err2 := apiutil.NewDiscoveryRESTMapper(cfg)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "failed to create mapper during gc")
		}
		return runtimeclient.New(cfg, runtimeclient.Options{
			Scheme: scheme.Scheme,
			Mapper: mapper,
		})
	}
	reconciler, err := NewMetaReconciler(client.New(mgr.GetClient()), mgr.GetCache(), mgr.GetConfig(), dc, clientFactory, startErrCh)
	if err != nil {
		return errors.Wrap(err, "could not create meta reconciler")
	}

	c, err := controller.New("metacontroller", mgr, controller.Options{
		Reconciler: reconciler,
	})
	if err != nil {
		return errors.Wrap(err, "could not create meta controller")
	}

	// Watch all changes to Syncs.
	if err = c.Watch(&source.Kind{Type: &nomosv1alpha1.Sync{}}, unaryHandler); err != nil {
		return errors.Wrap(err, "could not watch Syncs in the controller")
	}

	managerRestartCh := make(chan event.GenericEvent)
	managerRestartSource := &source.Channel{Source: managerRestartCh}
	if injectErr := managerRestartSource.InjectStopChannel(stopCh); injectErr != nil {
		return errors.Wrap(injectErr, "could not inject stop channel into genericResourceManager restart source")
	}
	// Create a watch for errors when starting the genericResourceManager and force a reconciliation.
	if err = c.Watch(managerRestartSource, unaryHandler); err != nil {
		return errors.Wrap(err, "could not watch manager initialization errors in the meta controller")
	}

	go func() {
		for {
			startErr := <-startErrCh
			if startErr != nil {
				// genericResourceManager could not successfully start, so we must clear its internal state before restarting.
				glog.Errorf("Error starting GenericResource controllers, restarting: %v", startErr)
				reconciler.genericResourceManager.Clear()
				// We always list all of the Syncs, so we can just send an empty event without a named resource.
				managerRestartCh <- event.GenericEvent{Meta: &metav1.ObjectMeta{}}
			}
		}
	}()

	return nil
}
