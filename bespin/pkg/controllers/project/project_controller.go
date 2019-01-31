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

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	"github.com/google/nomos/bespin/pkg/controllers/resource"
	"github.com/google/nomos/bespin/pkg/controllers/slices"
	"github.com/google/nomos/bespin/pkg/controllers/terraform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "project-controller"

// Add creates a new Project Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, ef terraform.ExecutorCreator) error {
	return add(mgr, newReconciler(mgr, ef))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, ef terraform.ExecutorCreator) reconcile.Reconciler {
	return &ReconcileProject{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		ef:       ef,
		recorder: mgr.GetRecorder(controllerName),
	}
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
	scheme   *runtime.Scheme
	ef       terraform.ExecutorCreator
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Project object and makes changes based on the state read
// and what is in the Project.Spec.
func (r *ReconcileProject) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), resource.ReconcileTimeout)
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
	tfe, err := r.ef.NewExecutor(ctx, r.Client, newProject)
	if err != nil {
		glog.Errorf("[Project %v] reconciler failed to create new Terraform executor: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to create new Terraform executor", name)
	}
	defer func() {
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[Project %v] reconciler failed to close Terraform executor: %v", name, cErr)
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
	done, err := resource.Update(ctx, r.Client, r.recorder, project, newProject)
	if err != nil {
		glog.Errorf("[Project %v] reconciler failed to update api server: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err, "[Project %v] reconciler failed to update api server", name)
	}
	if done {
		glog.V(1).Infof("[Project %v] reconciler successfully finished", name)
	}
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
	// The project is being deleted because its namespace is being deleted. k8s has an issue while deleting a
	// namespace containing resources with finalizers, the behavior is documented in b/123084662.
	// While deleting a namespace that contains a bespin Project resource, the project has a
	// bespin-defined finalizer that needs to be removed before the resource can be deleted. However
	// k8s in such case repeatedly updates the project's version/deletionTImestamp on api server, making
	// the project version on k8s api server diverged from the project that bespin holds, and this fails
	// bespin's Update request due to the "the object has been modified; please apply your changes to
	// the latest version and try again" error.
	// To address this the code below firstly reads the most update-to-date resource from k8s api server,
	// removes the finalizer, then writes back to k8s api server. The chances are
	// that the read-update-write takes minimal time (less than a second probably) during which the project
	// has not been updated again by k8s.
	// See https://github.com/kubernetes/kubernetes/issues/73098 for a long-term fix.
	pName := types.NamespacedName{Name: project.GetName(), Namespace: project.GetNamespace()}
	var err error
	for i := 0; i < resource.MaxRetries; i++ {
		ctx := context.Background()
		if err = r.Get(ctx, pName, project); err != nil {
			// Instance was just deleted.
			if k8serrors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			glog.Errorf("[Project %v] reconciler failed to get project instance: %v", pName, err)
			return reconcile.Result{}, err
		}
		project.ObjectMeta.Finalizers = slices.RemoveString(project.ObjectMeta.Finalizers, bespinv1.Finalizer)
		err = r.Update(ctx, project)
		if err == nil {
			return reconcile.Result{}, nil
		}
		glog.Errorf("[Project %v] reconciler failed to remove finalizer from k8s resource (retry count: %v): %v",
			pName, i, err)
	}
	return reconcile.Result{}, err
}
