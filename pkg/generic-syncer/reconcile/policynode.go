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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &PolicyNodeReconciler{}

// PolicyNodeReconciler reconciles a PolicyNode object.
type PolicyNodeReconciler struct {
	client client.Client
	cache  cache.Cache
	scheme *runtime.Scheme
}

// NewPolicyNodeReconciler returns a new PolicyNodeReconciler.
func NewPolicyNodeReconciler(client client.Client, cache cache.Cache, scheme *runtime.Scheme) *PolicyNodeReconciler {
	return &PolicyNodeReconciler{
		client: client,
		cache:  cache,
		scheme: scheme,
	}
}

// Reconcile is the Reconcile callback for PolicyNodeReconciler.
func (r *PolicyNodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	policyNode := &nomosv1.PolicyNode{}
	// TODO: pass in a valid context
	err := r.cache.Get(context.TODO(), request.NamespacedName, policyNode)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not actual, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		glog.Errorf("Could not find %q: %v", request, err)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	glog.Infof("policy node watched is: %v", policyNode)
	// TODO: reconcile actual namespace-scoped resource with declared policynode state.

	return reconcile.Result{}, nil
}
