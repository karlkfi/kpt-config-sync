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
	syncercache "github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	genericreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const clusterPolicyControllerName = "clusterpolicy-resources"

// AddClusterPolicy adds ClusterPolicy sync controllers to the Manager.
func AddClusterPolicy(mgr manager.Manager, decoder decode.Decoder,
	resourceTypes map[schema.GroupVersionKind]runtime.Object) error {
	genericClient := client.New(mgr.GetClient())
	applier, err := genericreconcile.NewApplier(mgr.GetConfig(), genericClient)
	if err != nil {
		return err
	}

	cpc, err := controller.New(clusterPolicyControllerName, mgr, controller.Options{
		Reconciler: genericreconcile.NewClusterPolicyReconciler(
			client.New(mgr.GetClient()),
			applier,
			syncercache.NewGenericResourceCache(mgr.GetCache()),
			mgr.GetRecorder(clusterPolicyControllerName),
			decoder,
			extractGVKs(resourceTypes),
		),
	})
	if err != nil {
		return errors.Wrapf(err, "could not create %q controller", clusterPolicyControllerName)
	}
	if err = cpc.Watch(&source.Kind{Type: &nomosv1.ClusterPolicy{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrapf(err, "could not watch ClusterPolicies in the %q controller", clusterPolicyControllerName)
	}

	mapToClusterPolicy := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToClusterPolicy),
	}
	// Set up a watch on all cluster-scoped resources defined in Syncs.
	// Look up the corresponding ClusterPolicy for the changed resources.
	for gvk, t := range resourceTypes {
		if err := cpc.Watch(&source.Kind{Type: t}, mapToClusterPolicy); err != nil {
			return errors.Wrapf(err, "could not watch %q in the %q controller", gvk, clusterPolicyControllerName)
		}
	}
	return nil
}

// genericResourceToClusterPolicy maps generic resources being watched,
// to reconciliation requests for the ClusterPolicy potentially managing them.
func genericResourceToClusterPolicy(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// There is only one ClusterPolicy potentially managing generic resources.
			Name: nomosv1.ClusterPolicyName,
		},
	}}
}
