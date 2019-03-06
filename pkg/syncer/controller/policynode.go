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

package controller

import (
	"context"

	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const policyNodeControllerName = "policynode-resources"

// AddPolicyNode adds PolicyNode sync controllers to the Manager.
func AddPolicyNode(ctx context.Context, mgr manager.Manager, decoder decode.Decoder,
	resourceTypes map[schema.GroupVersionKind]runtime.Object) error {
	genericClient := client.New(mgr.GetClient())
	applier, err := genericreconcile.NewApplier(mgr.GetConfig(), genericClient)
	if err != nil {
		return err
	}

	pnc, err := controller.New(policyNodeControllerName, mgr, controller.Options{
		Reconciler: genericreconcile.NewPolicyNodeReconciler(
			ctx,
			genericClient,
			applier,
			syncercache.NewGenericResourceCache(mgr.GetCache()),
			&CancelFilteringRecorder{mgr.GetRecorder(policyNodeControllerName)},
			decoder,
			extractGVKs(resourceTypes),
		),
	})
	if err != nil {
		return errors.Wrap(err, "could not create policynode controller")
	}

	mapValidPolicyNodes := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(validPolicyNodes),
	}
	if err = pnc.Watch(&source.Kind{Type: &nomosv1.PolicyNode{}}, mapValidPolicyNodes); err != nil {
		return errors.Wrapf(err, "could not watch PolicyNodes in the %q controller", policyNodeControllerName)
	}
	// Namespaces have the same name as their corresponding PolicyNode.
	if err = pnc.Watch(&source.Kind{Type: &corev1.Namespace{}}, mapValidPolicyNodes); err != nil {
		return errors.Wrapf(err, "could not watch Namespaces in the %q controller", policyNodeControllerName)
	}

	maptoPolicyNode := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToPolicyNode),
	}
	// Set up a watch on all namespace-scoped resources defined in Syncs.
	// Look up the corresponding PolicyNode for the changed resources.
	for gvk, t := range resourceTypes {
		if err := pnc.Watch(&source.Kind{Type: t}, maptoPolicyNode); err != nil {
			return errors.Wrapf(err, "could not watch %q in the %q controller", gvk, policyNodeControllerName)
		}
	}
	return nil
}

// genericResourceToPolicyNode maps generic resources being watched,
// to reconciliation requests for the PolicyNode potentially managing them.
func genericResourceToPolicyNode(o handler.MapObject) []reconcile.Request {
	return reconcileRequests(o.Meta.GetNamespace())
}

// validPolicyNodes returns reconcile requests for the watched objects,
// if the name of the resource corresponds to a PolicyNode that can be synced.
func validPolicyNodes(o handler.MapObject) []reconcile.Request {
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
