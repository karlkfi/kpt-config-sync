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
	"github.com/google/nomos/pkg/bespin-controllers/slices"
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
// The comment line below(starting with +kubebuilder) does not work without kubebuilder code layout. It was
// created by kubebuilder in some other repo. Kubebuilder can parse it to generate RBAC YAML.
// +kubebuilder:rbac:groups=bespin.dev,resources=projects,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileProject) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	project := &bespinv1.Project{}
	if err := r.Get(ctx, request.NamespacedName, project); err != nil {
		// Instance was just deleted or there's some internal K8S error.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("[Project %v] failed to get project instance: %v", request.NamespacedName, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] failed to get project instance", request.NamespacedName)
	}
	// TODO(b/119327784): Handle the deletion by using finalizer: check for deletionTimestamp, verify
	// the delete finalizer is there, handle delete from GCP, then remove the finalizer.
	tfe, err := terraform.NewExecutor(ctx, r.Client, project)
	if err != nil {
		glog.Errorf("[Project %v] reconciler failed to create new Terraform executor: %v", request.NamespacedName, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to create new Terraform executor", request.NamespacedName)
	}
	defer func() {
		if err = tfe.Close(); err != nil {
			glog.Errorf("[Project %v] reconciler failed to close Terraform executor: %v", request.NamespacedName, err)
			err = errors.Wrapf(err,
				"[Project %v] reconciler failed to close Terraform executor", request.NamespacedName)
		}
	}()
	// Project is being deleted.
	if !project.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.doDeletion(tfe, project)
	}
	// Project is not being deleted, make sure it has bespinv1.Finalizer.
	if !slices.ContainsString(project.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		project.ObjectMeta.Finalizers = append(project.ObjectMeta.Finalizers, bespinv1.Finalizer)
		if err = r.Update(context.Background(), project); err != nil {
			return reconcile.Result{Requeue: true}, nil
		}
	}
	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		glog.Errorf("[Project %v] reconciler failed to run Terraform command: %v", request.NamespacedName, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to run Terraform command", request.NamespacedName)
	}
	if err = r.updateServer(ctx, project); err != nil {
		err = errors.Wrap(err, "reconciler failed to update Project in API server")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// doDeletion deletes the Project on GCP via Terraform, and removes finalizer so that the Project resource on k8s API
// server will be deleted as well.
func (r *ReconcileProject) doDeletion(tfe *terraform.Executor, project *bespinv1.Project) (reconcile.Result, error) {
	// Project has our finalizer, so run terraform flow to delete resource.
	if !slices.ContainsString(project.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		glog.Warningf("[Project %v] instance being deleted does not have bespin finalizer.", project.Spec.Name)
	}
	if err := tfe.RunDeleteFlow(); err != nil {
		glog.Errorf("[Project %v] reconciler failed to run Terraform command in project deletion: %v",
			project.Spec.Name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Project %v] reconciler failed to run Terraform command in project deletion.", project.Spec.Name)
	}
	// Remove bespinv1.Finalizer after deletion so k8s resource can be removed.
	project.ObjectMeta.Finalizers = slices.RemoveString(project.ObjectMeta.Finalizers, bespinv1.Finalizer)
	if err := r.Update(context.Background(), project); err != nil {
		glog.Errorf("[Project %v] reconciler failed to remove finalizer from k8s resource: %v",
			project.Spec.Name, err)
		return reconcile.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, nil
}

// updateServer updates the Project object in k8s API server.
// Note: r.Update() will trigger another Reconcile(), we should't update the API server
// when there is nothing changed.
func (r *ReconcileProject) updateServer(ctx context.Context, p *bespinv1.Project) error {
	newP := &bespinv1.Project{}
	p.DeepCopyInto(newP)
	newP.Status.SyncDetails.Token = p.Spec.ImportDetails.Token
	newP.Status.SyncDetails.Error = ""

	if equality.Semantic.DeepEqual(p, newP) {
		glog.V(1).Infof("[Project %v] nothing to update", newP.Spec.Name)
		return nil
	}
	newP.Status.SyncDetails.Time = metav1.Now()
	if err := r.Update(ctx, newP); err != nil {
		return errors.Wrap(err, "failed to update Project in API server")
	}
	return nil
}
