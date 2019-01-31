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

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	"github.com/google/nomos/bespin/pkg/controllers/resource"
	"github.com/google/nomos/bespin/pkg/controllers/slices"
	"github.com/google/nomos/bespin/pkg/controllers/terraform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "clusteriampolicy-controller"

// Add creates a new Folder Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, ef terraform.ExecutorCreator) error {
	return add(mgr, newReconciler(mgr, ef))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, ef terraform.ExecutorCreator) reconcile.Reconciler {
	return &ReconcileFolder{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		ef:       ef,
		recorder: mgr.GetRecorder(controllerName),
	}
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
	scheme   *runtime.Scheme
	ef       terraform.ExecutorCreator
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Folder object and makes changes based on the state read
// and what is in the Folder.Spec. In cases where the underlying Terraform commands return errors, the error
// details will be updated in the k8s resource "Status.SyncDetails.Error" field and the request will be
// retried.
func (r *ReconcileFolder) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), resource.ReconcileTimeout)
	defer cancel()
	name := request.NamespacedName
	folder := &bespinv1.Folder{}
	if err := r.Get(ctx, request.NamespacedName, folder); err != nil {
		// Instance was just deleted.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("[Folder %v] reconciler failed to get folder instance: %v", name, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Folder %v] reconciler failed to get folder instance", name)
	}
	newFolder := &bespinv1.Folder{}
	folder.DeepCopyInto(newFolder)
	tfe, err := r.ef.NewExecutor(ctx, r.Client, newFolder)
	if err != nil {
		glog.Errorf("[Folder %v] reconciler failed to create new terraform executor: %v", name, err)
		return reconcile.Result{},
			errors.Wrapf(err, "[Folder %v] reconciler failed to create new terraform executor", name)
	}
	defer func() {
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[Folder %v] reconciler failed to close Terraform executor: %v", name, cErr)
		}
	}()
	// Folder has been requested for deletion.
	if !newFolder.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.doDeletion(tfe, newFolder)
	}
	// Folder is not being deleted, make sure it has bespinv1.Finalizer.
	if !slices.ContainsString(newFolder.ObjectMeta.Finalizers, bespinv1.Finalizer) {
		newFolder.ObjectMeta.Finalizers = append(newFolder.ObjectMeta.Finalizers, bespinv1.Finalizer)
		if err = r.Update(context.Background(), newFolder); err != nil {
			glog.Errorf("[Folder %v] reconciler failed to add finalizer to k8s resource: ", err)
			err = errors.Wrapf(err, "[Folder %v] reconciler failed to add finalizer to k8s resource", name)
		}
		return reconcile.Result{}, err
	}
	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		glog.Errorf("[Folder %v] reconciler failed to execute Terraform commands: %v", name, err)
		err = errors.Wrapf(err, "[Folder %v] reconciler failed to execute Terraform commands", name)
		// TODO(b/123044952): populate the error message to resource status.
		return reconcile.Result{}, err
	}

	// Update the Folder ID field in Spec:
	// 1. If this is a new Folder, adding the ID will allow its child-folder or child-projects
	//    to be created under it.
	// 2. If this is an existing Folder, the Folder ID effectively is not changed.
	if err = updateFolderID(tfe, newFolder); err != nil {
		glog.Errorf("[Folder %v] reconciler failed to update Folder ID: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err, "[Folder %v] reconciler failed to update Folder ID", name)
	}
	done, err := resource.Update(ctx, r.Client, r.recorder, folder, newFolder)
	if err != nil {
		glog.Errorf("[Folder %v] reconciler failed to update api server: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err, "[Folder %v] reconciler failed to update api server", name)
	}
	if done {
		glog.V(1).Infof("[Folder %v] reconciler successfully finished", name)
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

func updateFolderID(tfe *terraform.Executor, folder *bespinv1.Folder) error {
	if err := tfe.UpdateState(); err != nil {
		return errors.Wrapf(err, "[Folder %v] failed to update terraform state", folder.Spec.DisplayName)
	}
	id, err := tfe.GetFolderID()
	if err != nil {
		return errors.Wrapf(err, "[Folder %v] failed to get Folder ID from terraform state", folder.Spec.DisplayName)
	}
	folder.Spec.ID = id
	return nil
}
