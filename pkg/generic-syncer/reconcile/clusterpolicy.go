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

package reconcile

import (
	"context"

	"github.com/golang/glog"
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	syncercache "github.com/google/nomos/pkg/generic-syncer/cache"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &ClusterPolicyReconciler{}

// ClusterPolicyReconciler reconciles a ClusterPolicy object.
type ClusterPolicyReconciler struct {
	client client.Client
	cache  syncercache.GenericCache
}

// NewClusterPolicyReconciler returns a new ClusterPolicyReconciler.
func NewClusterPolicyReconciler(client client.Client, cache syncercache.GenericCache) *ClusterPolicyReconciler {
	return &ClusterPolicyReconciler{
		client: client,
		cache:  cache,
	}
}

// Reconcile is the Reconcile callback for ClusterPolicyReconciler.
func (r *ClusterPolicyReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	clusterPolicy := &nomosv1.ClusterPolicy{}
	// TODO: pass in a valid context
	err := r.cache.Get(context.TODO(), request.NamespacedName, clusterPolicy)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not actual, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	glog.Infof("clusterpolicy watched is: %v", clusterPolicy)
	// TODO: reconcile actual cluster-scoped resource with declared clusterpolicy state.

	return reconcile.Result{}, nil
}
