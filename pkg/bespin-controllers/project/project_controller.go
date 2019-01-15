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

package project

import (
	"context"
	"time"

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/resource"
	"github.com/google/nomos/pkg/bespin-controllers/slices"
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const reconcileTimeout = time.Minute * 5

// Add creates a new Project Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileProject{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller.
	c, err := controller.New("project-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrap(err, "failed to create new Project controller")
	}

	// Watch for changes to Project.
	err = c.Watch(&source.Kind{Type: &bespinv1.Project{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return errors.Wrap(err, "failed to watch Project")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileProject{}

// ReconcileProject reconciles a Project object.
type ReconcileProject struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Project object and makes changes based on the state read
// and what is in the Project.Spec.
func (r *ReconcileProject) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	name := request.NamespacedName
	project := &bespinv1.Project{}
	if err := r.Get(ctx, name, project); err != nil {
		// Instance was just deleted.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("[Project %v] failed to get project instance: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] failed to get project instance", name)
	}
	newProject := &bespinv1.Project{}
	project.DeepCopyInto(newProject)
	tfe, err := terraform.NewExecutor(ctx, r.Client, newProject)
	if err != nil {
		glog.Errorf("[Project %v] reconciler failed to create new Terraform executor: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to create new Terraform executor", name)
	}
	defer func() {
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[Folder %v] reconciler failed to close Terraform executor: %v", name, cErr)
		}
	}()
	// Project is being deleted.
	if !newProject.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.doDeletion(tfe, newProject)
	}
	// Project is not being deleted, make sure it has bespinv1.Finalizer.
	if !slices.ContainsString(newProject.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		newProject.ObjectMeta.Finalizers = append(newProject.ObjectMeta.Finalizers, bespinv1.Finalizer)
		if err = r.Update(context.Background(), newProject); err != nil {
			glog.Errorf("[Project %v] reconciler failed to add finalizer to k8s resource: ", err)
			err = errors.Wrapf(err, "[Project %v] reconciler failed to add finalizer to k8s resource", name)
		}
		return reconcile.Result{}, err
	}
	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		glog.Errorf("[Project %v] reconciler failed to run Terraform command: %v", name, err)
		// TODO(b/123044952): populate the error message to resource status.
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to run Terraform command", name)
	}
	if err = resource.Update(ctx, r.Client, project, newProject); err != nil {
		glog.Errorf("[Project %v] reconciler failed to update Project in API server: %v", name, err)
		err = errors.Wrap(err, "reconciler failed to update Project in API server")
		return reconcile.Result{}, err
	}
	glog.V(1).Infof("[Project %v] reconciler successfully finished", name)
	return reconcile.Result{}, nil
}

// doDeletion deletes the Project on GCP via Terraform, and removes finalizer so that the Project resource on k8s API
// server will be deleted as well.
func (r *ReconcileProject) doDeletion(tfe *terraform.Executor, project *bespinv1.Project) (reconcile.Result, error) {
	// Project has our finalizer, so run terraform flow to delete resource.
	if !slices.ContainsString(project.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		glog.Warningf("[Project %v] instance being deleted does not have bespin finalizer.", project.Spec.DisplayName)
	}
	if err := tfe.RunDeleteFlow(); err != nil {
		glog.Errorf("[Project %v] reconciler failed to run Terraform command in project deletion: %v",
			project.Spec.DisplayName, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to run Terraform command in project deletion.", project.Spec.DisplayName)
	}
	// Remove bespinv1.Finalizer after deletion so k8s resource can be removed.
	project.ObjectMeta.Finalizers = slices.RemoveString(project.ObjectMeta.Finalizers, bespinv1.Finalizer)
	if err := r.Update(context.Background(), project); err != nil {
		glog.Errorf("[Project %v] reconciler failed to remove finalizer from k8s resource: %v",
			project.Spec.DisplayName, err)
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, nil
}
