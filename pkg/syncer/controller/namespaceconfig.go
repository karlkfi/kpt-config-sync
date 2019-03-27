/*
Copyright 2018 The CSP Config Management Authors.
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

package controller

import (
	"context"

	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	syncercache "github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	genericreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const namespaceConfigControllerName = "namespaceconfig-resources"

// AddNamespaceConfig adds NamespaceConfig sync controllers to the Manager.
func AddNamespaceConfig(ctx context.Context, mgr manager.Manager, decoder decode.Decoder,
	resourceTypes map[schema.GroupVersionKind]runtime.Object) error {
	genericClient := client.New(mgr.GetClient())
	applier, err := genericreconcile.NewApplier(mgr.GetConfig(), genericClient)
	if err != nil {
		return err
	}

	pnc, err := k8scontroller.New(namespaceConfigControllerName, mgr, k8scontroller.Options{
		Reconciler: genericreconcile.NewNamespaceConfigReconciler(
			ctx,
			genericClient,
			applier,
			syncercache.NewGenericResourceCache(mgr.GetCache()),
			&CancelFilteringRecorder{mgr.GetRecorder(namespaceConfigControllerName)},
			decoder,
			extractGVKs(resourceTypes),
		),
	})
	if err != nil {
		return errors.Wrap(err, "could not create namespaceconfig controller")
	}

	mapValidNamespaceConfigs := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(validNamespaceConfigs),
	}
	if err = pnc.Watch(&source.Kind{Type: &nomosv1.NamespaceConfig{}}, mapValidNamespaceConfigs); err != nil {
		return errors.Wrapf(err, "could not watch NamespaceConfigs in the %q controller", namespaceConfigControllerName)
	}
	// Namespaces have the same name as their corresponding NamespaceConfig.
	if err = pnc.Watch(&source.Kind{Type: &corev1.Namespace{}}, mapValidNamespaceConfigs); err != nil {
		return errors.Wrapf(err, "could not watch Namespaces in the %q controller", namespaceConfigControllerName)
	}

	maptoNamespaceConfig := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToNamespaceConfig),
	}
	// Set up a watch on all namespace-scoped resources defined in Syncs.
	// Look up the corresponding NamespaceConfig for the changed resources.
	for gvk, t := range resourceTypes {
		if err := pnc.Watch(&source.Kind{Type: t}, maptoNamespaceConfig); err != nil {
			return errors.Wrapf(err, "could not watch %q in the %q controller", gvk, namespaceConfigControllerName)
		}
	}
	return nil
}

// genericResourceToNamespaceConfig maps generic resources being watched,
// to reconciliation requests for the NamespaceConfig potentially managing them.
func genericResourceToNamespaceConfig(o handler.MapObject) []reconcile.Request {
	return reconcileRequests(o.Meta.GetNamespace())
}

// validNamespaceConfigs returns reconcile requests for the watched objects,
// if the name of the resource corresponds to a NamespaceConfig that can be synced.
func validNamespaceConfigs(o handler.MapObject) []reconcile.Request {
	return reconcileRequests(o.Meta.GetName())
}

func reconcileRequests(ns string) []reconcile.Request {
	if namespaceutil.IsReserved(ns) {
		// Ignore changes to resources outside a namespace Nomos can manage.
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// The namespace of the generic resource is the name of the policy node potentially managing it.
			Name: ns,
		},
	}}
}
