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
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	syncercache "github.com/google/nomos/pkg/generic-syncer/cache"
	"github.com/google/nomos/pkg/generic-syncer/client"
	"github.com/google/nomos/pkg/generic-syncer/decode"
	"github.com/google/nomos/pkg/generic-syncer/differ"
	genericreconcile "github.com/google/nomos/pkg/generic-syncer/reconcile"
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
func AddPolicyNode(mgr manager.Manager, decoder decode.Decoder, comparator *differ.Comparator,
	resourceTypes map[schema.GroupVersionKind]runtime.Object) error {
	genericClient := client.New(mgr.GetClient())
	applier, err := genericreconcile.NewApplier(mgr.GetScheme(), mgr.GetConfig(), genericClient)
	if err != nil {
		return err
	}

	pnc, err := controller.New(policyNodeControllerName, mgr, controller.Options{
		Reconciler: genericreconcile.NewPolicyNodeReconciler(
			genericClient,
			applier,
			syncercache.NewGenericResourceCache(mgr.GetCache()),
			mgr.GetRecorder(policyNodeControllerName),
			decoder,
			comparator,
			extractGVKs(resourceTypes),
		),
	})
	if err != nil {
		return errors.Wrap(err, "could not create policynode controller")
	}

	if err = pnc.Watch(&source.Kind{Type: &nomosv1.PolicyNode{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrap(err, "could not watch PolicyNodes in the policy node controller")
	}
	// Namespaces have the same name as their corresponding PolicyNode, so we can also use EnqueueRequestForObject.
	if err = pnc.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrap(err, "could not watch Namespaces in the policy node controller")
	}

	maptoPolicyNode := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToPolicyNode),
	}
	// Set up a watch on all namespace-scoped resources defined in Syncs.
	// Look up the corresponding PolicyNode for the changed resources.
	for gvk, t := range resourceTypes {
		if err := pnc.Watch(&source.Kind{Type: t}, maptoPolicyNode); err != nil {
			return errors.Wrapf(err, "could not watch %q in the generic controller", gvk)
		}
	}
	return nil
}

// genericResourceToPolicyNode maps generic resources being watched,
// to reconciliation requests for the PolicyNode potentially managing them.
func genericResourceToPolicyNode(o handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// The namespace of the generic resource is the name of the policy node potentially managing it.
			Name: o.Meta.GetNamespace(),
		},
	}}
}
