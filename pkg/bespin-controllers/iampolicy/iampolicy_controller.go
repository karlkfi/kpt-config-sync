/*
Copyright 2018 Google LLC.

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

package iampolicy

import (
	"context"
	"time"

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const reconcileTimeout = time.Minute * 5

// Add creates a new IAMPolicy Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileIAMPolicy{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller.
	c, err := controller.New("iampolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrap(err, "failed to create new IAMPolicy controller")
	}

	// Watch for changes to IAMPolicy.
	err = c.Watch(&source.Kind{Type: &bespinv1.IAMPolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return errors.Wrap(err, "failed to watch IAMPolicy")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIAMPolicy{}

// ReconcileIAMPolicy reconciles a IAMPolicy object.
type ReconcileIAMPolicy struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a IAMPolicy object and makes changes based on the state read
// and what is in the IAMPolicy.Spec.
// The comment line below (starting with +kubebuilder) does not work without kubebuilder code layout. It was
// created by kubebuilder in some other repo. Kubebuilder can parse it to generate RBAC YAML.
// +kubebuilder:rbac:groups=bespin.dev,resources=iampolicy,verbs=get;list;watch;create;update;patch;delete
// TODO(b/120504718): Refactor the code to avoid using string literals.
func (r *ReconcileIAMPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the IAMPolicy instance
	instance := &bespinv1.IAMPolicy{}
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		glog.Errorf("IAMPolicy reconciler error in getting iampolicy instance: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler error in getting iampolicy instance")
	}
	// TODO(b/119327784): Handle the deletion by using finalizer: check for deletionTimestamp, verify
	// the delete finalizer is there, handle delete from GCP, then remove the finalizer.
	tfe, err := terraform.NewExecutor(instance)
	if err != nil {
		glog.Errorf("IAMPolicy reconciler failed to create new Terraform executor: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler failed to create new Terraform executor")
	}
	defer func() {
		err = tfe.Close()
		if err != nil {
			glog.Errorf("IAMPolicy reconciler failed to close Terraform executor: %v", err)
			err = errors.Wrap(err, "IAMPolicy reconciler failed to close Terraform executor")
		}
	}()

	err = tfe.RunCreateOrUpdateFlow()
	if err != nil {
		glog.Errorf("IAMPolicy reconciler failed to run Terraform command: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler failed to run Terraform command")
	}

	return reconcile.Result{}, nil
}
