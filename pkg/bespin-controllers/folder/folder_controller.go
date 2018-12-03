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

package folder

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

// Add creates a new Folder Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileFolder{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("folder-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to folder
	err = c.Watch(&source.Kind{Type: &bespinv1.Folder{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return errors.Wrap(err, "folder controller error in adding instance to watch")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileFolder{}

// ReconcileFolder reconciles a Folder object.
type ReconcileFolder struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Folder object and makes changes based on the state read
// and what is in the Folder.Spec.
// The comment line below(starting with +kubebuilder) does not work without kubebuilder code layout. It was
// created by kubebuilder in some other repo. Kubebuilder can parse it to generate rbac yaml.
// +kubebuilder:rbac:groups=bespin.dev,resources=folders,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileFolder) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the folder instance
	instance := &bespinv1.Folder{}
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		glog.Errorf("folder reconciler error in getting folder instance: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "folder reconciler error in getting folder instance")
	}
	// TODO(b/119327784): Handle the deletion by using finalizer: check for deletionTimestamp, verify
	// the delete finalizer is there, handle delete from GCP, then remove the finalizer.
	tfe, err := terraform.NewExecutor(instance)
	if err != nil {
		glog.Errorf("folder reconciler failed to create new terraform executor: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "folder reconciler failed to create new terraform executor")
	}
	defer func() {
		err = tfe.Close()
		if err != nil {
			glog.Errorf("folder reconciler failed to close Terraform executor: %v", err)
			err = errors.Wrap(err, "folder reconciler failed to close Terraform executor")
		}
	}()

	err = tfe.RunAll()
	if err != nil {
		glog.Errorf("Folder reconciler failed to run Terraform command: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "Folder reconciler failed to run Terraform command")
	}

	// Update back the Folder ID to Spec.

	return reconcile.Result{}, nil
}
