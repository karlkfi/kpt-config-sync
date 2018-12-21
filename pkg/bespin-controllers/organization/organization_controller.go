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

package organization

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

// Add creates a new Organization Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileOrganization{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("organization-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to organization
	err = c.Watch(&source.Kind{Type: &bespinv1.Organization{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return errors.Wrap(err, "organization controller error in adding instance to watch")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileOrganization{}

// ReconcileOrganization reconciles a Organization object.
type ReconcileOrganization struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Organization object and makes sure the Organization exists
// on GCP. Organizations are different from other GCP resources they are not allowed to be created,
// updated (OrgPolicy can be attached, but not update the Organization itself), deleted.
// In cases where the underlying Terraform commands return errors, the error
// details will be updated in the k8s resource "Status.SyncDetails.Error" field and the request will be
// retried.
// The comment line below(starting with +kubebuilder) does not work without kubebuilder code layout. It was
// created by kubebuilder in some other repo. Kubebuilder can parse it to generate rbac yaml.
// +kubebuilder:rbac:groups=bespin.dev,resources=organizations,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileOrganization) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	organization := &bespinv1.Organization{}
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	err := r.Get(ctx, request.NamespacedName, organization)
	if err != nil {
		glog.Errorf("[Organization %v] reconciler failed to get organization instance: %v", request.NamespacedName, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Organization %v] reconciler failed to get organization instance", request.NamespacedName)
	}
	tfe, err := terraform.NewExecutor(ctx, r.Client, organization)
	if err != nil {
		glog.Errorf("[Organization %v] reconciler failed to create new Terraform executor: %v", request.NamespacedName, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Organization %v] reconciler failed to create new Terraform executor", request.NamespacedName)
	}
	defer func() {
		if err != nil {
			glog.Errorf("[Organization %v] reconciler failed: %v", request.NamespacedName, err)
			return
		}
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[Organization %v] reconciler failed to close Terraform executor: %v", request.NamespacedName, cErr)
		}
	}()

	// No organization CRUD supports at this moment, only need to make sure the Organization exists in GCP.
	if err = tfe.RunInit(); err != nil {
		return reconcile.Result{}, err
	}
	if err = tfe.RunPlan(); err != nil {
		return reconcile.Result{}, err
	}
	// TODO(b/120977710): Currently Organization Status is empty, should we populate back "READY"?
	return reconcile.Result{}, nil
}
