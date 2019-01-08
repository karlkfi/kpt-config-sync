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
// and what is in the Folder.Spec. In cases where the underlying Terraform commands return errors, the error
// details will be updated in the k8s resource "Status.SyncDetails.Error" field and the request will be
// retried.
// The comment line below(starting with +kubebuilder) does not work without kubebuilder code layout. It was
// created by kubebuilder in some other repo. Kubebuilder can parse it to generate rbac yaml.
// +kubebuilder:rbac:groups=bespin.dev,resources=folders,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileFolder) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	folder := &bespinv1.Folder{}
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	if err := r.Get(ctx, request.NamespacedName, folder); err != nil {
		// Instance was just deleted.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("[Folder %v] reconciler failed to get folder instance: %v", request.NamespacedName, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Folder %v] reconciler failed to get folder instance", request.NamespacedName)
	}
	if err := folder.Validate(); err != nil {
		glog.Errorf("[Folder %v] reconciler failed to validate Folder instance: %v", request.NamespacedName, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Folder %v] reconciler failed to validate Folder instance", request.NamespacedName)
	}
	tfe, err := terraform.NewExecutor(ctx, r.Client, folder)
	if err != nil {
		glog.Errorf("[Folder %v] reconciler failed to create new terraform executor: %v", request.NamespacedName, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Folder %v] reconciler failed to create new terraform executor", request.NamespacedName)
	}
	defer func() {
		if err != nil {
			glog.Errorf("[Folder %v] reconciler failed: %v", request.NamespacedName, err)
		}
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[Folder %v] reconciler failed to close Terraform executor: %v", request.NamespacedName, cErr)
		}
	}()
	// Folder has been requested for deletion.
	if !folder.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.doDeletion(tfe, folder)
	}
	// Folder is not being deleted, make sure it has bespinv1.Finalizer.
	if !slices.ContainsString(folder.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		folder.ObjectMeta.Finalizers = append(folder.ObjectMeta.Finalizers, bespinv1.Finalizer)
		if err = r.Update(context.Background(), folder); err != nil {
			glog.Errorf("[Folder %v] reconciler failed to add finalizer to k8s resource: %v",
				folder.Spec.DisplayName, err)
			return reconcile.Result{Requeue: true}, nil
		}
	}
	// If Terraform returns an error, update API server with the error details; otherwise update
	// the API server to bring the resource's Status in sync with its Spec.
	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		err = errors.Wrapf(err, "[Folder %v] reconciler failed to execute Terraform commands", request.NamespacedName)
		folder.Status.SyncDetails.Error = err.Error()
		if uErr := r.Update(ctx, folder); uErr != nil {
			err = errors.Wrapf(err, "[Folder %v] reconciler failed to update Folder in API server: %v",
				request.NamespacedName, uErr)
		}
		return reconcile.Result{}, err
	}
	if err = r.updateAPIServer(ctx, tfe, folder); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "[Folder %v] reconciler failed to update Folder in API server",
			request.NamespacedName)
	}
	return reconcile.Result{}, nil
}

// doDeletion deletes the Folder on GCP via Terraform, and removes finalizer so that the Folder resource on k8s API
// server will be deleted as well.
func (r *ReconcileFolder) doDeletion(tfe *terraform.Executor, folder *bespinv1.Folder) (reconcile.Result, error) {
	if !slices.ContainsString(folder.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		glog.Warningf("[Folder %v] instance being deleted does not have bespin finalizer.", folder.Spec.DisplayName)
	}
	if err := tfe.RunDeleteFlow(); err != nil {
		glog.Errorf("[Folder %v] reconciler failed to run Terraform command in folder deletion: %v",
			folder.Spec.DisplayName, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[Folder %v] reconciler failed to run Terraform command in folder deletion.", folder.Spec.DisplayName)
	}
	// Remove bespinv1.Finalizer after deletion so k8s resource can be removed.
	folder.ObjectMeta.Finalizers = slices.RemoveString(folder.ObjectMeta.Finalizers, bespinv1.Finalizer)
	if err := r.Update(context.Background(), folder); err != nil {
		glog.Errorf("[Folder %v] reconciler failed to remove finalizer from k8s resource: %v",
			folder.Spec.DisplayName, err)
		return reconcile.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, nil
}

// updateAPIServer updates the Folder object in k8s API server.
// Note: r.Update() will trigger another Reconcile(), we should't update the API server
// when there is nothing changed.
func (r *ReconcileFolder) updateAPIServer(ctx context.Context, tfe *terraform.Executor, f *bespinv1.Folder) error {
	if err := tfe.UpdateState(); err != nil {
		return errors.Wrapf(err, "[Folder %v] failed to update terraform state", f.Spec.DisplayName)
	}
	id, err := tfe.GetFolderID()
	if err != nil {
		return errors.Wrapf(err, "[Folder %v] failed to get Folder ID from terraform state", f.Spec.DisplayName)
	}

	newF := &bespinv1.Folder{}
	f.DeepCopyInto(newF)
	newF.Status.ID = id
	newF.Status.SyncDetails.Token = f.Spec.ImportDetails.Token
	newF.Status.SyncDetails.Error = ""

	if equality.Semantic.DeepEqual(f, newF) {
		glog.V(1).Infof("[Folder %v] nothing to update", newF.Spec.DisplayName)
		return nil
	}
	newF.Status.SyncDetails.Time = metav1.Now()
	if err = r.Update(ctx, newF); err != nil {
		return errors.Wrapf(err, "[Folder %v] failed to update Folder in API server", newF.Spec.DisplayName)
	}
	return nil
}
