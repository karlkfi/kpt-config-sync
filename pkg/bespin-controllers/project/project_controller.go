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
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/pkg/errors"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
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
	project := &bespinv1.Project{}
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	err := r.Get(ctx, request.NamespacedName, project)
	if err != nil {
		glog.Errorf("[Project %v] reconciler failed to get Project instance: %v", request.NamespacedName, err)
		return reconcile.Result{}, errors.Wrapf(err, "[Project %v] reconciler failed to get Project instance", request.NamespacedName)
	}
	// TODO(b/119327784): Handle the deletion by using finalizer: check for deletionTimestamp, verify
	// the delete finalizer is there, handle delete from GCP, then remove the finalizer.
	tfe, err := terraform.NewExecutor(project)
	if err != nil {
		glog.Errorf("[Project %v] reconciler failed to create new terraform executor: %v", request.NamespacedName, err)
		return reconcile.Result{}, errors.Wrapf(err, "[Project %v] reconciler failed to create new terraform executor", request.NamespacedName)
	}
	defer func() {
		if err != nil {
			glog.Errorf("[Project %v] reconciler failed: %v", request.NamespacedName, err)
		}
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[Project %v] reconciler failed to close Terraform executor: %v", request.NamespacedName, cErr)
		}
	}()

	// If Terraform returns an error, update API server with the error details; otherwise update
	// the API server to bring the resource's Status in sync with its Spec.
	if err = tfe.RunAll(); err != nil {
		err = errors.Wrapf(err, "reconciler failed to execute Terraform commands")
		project.Status.SyncDetails.Error = err.Error()
		if uErr := r.Update(ctx, project); uErr != nil {
			err = errors.Wrapf(err, "reconciler failed to update Project in API server: %v", uErr)
		}
		return reconcile.Result{}, err
	}

	if err = r.updateAPIServer(ctx, project); err != nil {
		err = errors.Wrap(err, "reconciler failed to update Project in API server")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// updateAPIServer updates the Project object in k8s API server.
// Note: r.Update() will trigger another Reconcile(), we should't update the API server
// when there is nothing changed.
func (r *ReconcileProject) updateAPIServer(ctx context.Context, p *bespinv1.Project) error {
	newP := &bespinv1.Project{}
	p.DeepCopyInto(newP)
	newP.Status.SyncDetails.Token = p.Spec.ImportDetails.Token
	newP.Status.SyncDetails.Error = ""

	if apiequality.Semantic.DeepEqual(p, newP) {
		glog.V(1).Infof("[Project %v] nothing to update", newP.Spec.Name)
		return nil
	}
	newP.Status.SyncDetails.Time = metav1.Now()
	if err := r.Update(ctx, newP); err != nil {
		return errors.Wrap(err, "failed to update Project in API server")
	}
	return nil
}
