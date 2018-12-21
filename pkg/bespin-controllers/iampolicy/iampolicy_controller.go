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
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	// Fetch the IAMPolicy instance.
	iampolicy := &bespinv1.IAMPolicy{}
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	if err := r.Get(ctx, request.NamespacedName, iampolicy); err != nil {
		// Instance was just deleted or there's some internal K8S error.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("IAMPolicy reconciler error in getting iampolicy instance: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler error in getting iampolicy instance")
	}
	if err := r.ensureOwnerReference(ctx, iampolicy); err != nil {
		glog.Errorf("IAMPolicy reconciler failed to set owner reference: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler failed to set owner reference")
	}
	// TODO(b/119327784): Handle the deletion by using finalizer: check for deletionTimestamp, verify
	// the delete finalizer is there, handle delete from GCP, then remove the finalizer.
	tfe, err := terraform.NewExecutor(ctx, r.Client, iampolicy)
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

	if err = r.updateServer(ctx, iampolicy); err != nil {
		err = errors.Wrap(err, "reconciler failed to update IAMPolicy in API server")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileIAMPolicy) ensureOwnerReference(ctx context.Context, iampolicy *bespinv1.IAMPolicy) error {
	if iampolicy.Spec.ResourceReference.Kind != bespinv1.ProjectKind {
		return errors.Errorf("invalid resource reference reference kind: %v", iampolicy.Spec.ResourceReference.Kind)
	}
	resourceName := types.NamespacedName{Namespace: iampolicy.Namespace, Name: iampolicy.Spec.ResourceReference.Name}
	project := &bespinv1.Project{}
	if err := r.Get(ctx, resourceName, project); err != nil {
		return errors.Wrapf(err, "failed to get resource reference Project instance: %v", resourceName)
	}
	uid := project.GetUID()
	if uid == "" {
		return errors.Errorf("missing resource reference Project UID: %v", resourceName)
	}
	name := project.GetName()
	if name == "" {
		return errors.Errorf("missing resource reference Project Name: %v", resourceName)
	}
	owner := metav1.OwnerReference{
		Kind:       bespinv1.ProjectKind,
		APIVersion: bespinv1.SchemeGroupVersion.Version,
		Name:       name,
		UID:        uid,
	}
	glog.V(1).Infof("[IAMPolicy %v] set OwnerReference: %v", iampolicy.Name, owner)
	iampolicy.SetOwnerReferences([]metav1.OwnerReference{owner})
	return nil
}

// updateServer updates the IAMPolicy object in k8s API server.
// Note: r.Update() will trigger another Reconcile(), we should't update the API server
// when there is nothing changed.
func (r *ReconcileIAMPolicy) updateServer(ctx context.Context, iampolicy *bespinv1.IAMPolicy) error {
	newI := &bespinv1.IAMPolicy{}
	iampolicy.DeepCopyInto(newI)
	newI.Status.SyncDetails.Token = iampolicy.Spec.ImportDetails.Token
	newI.Status.SyncDetails.Error = ""

	// If there's no diff, we don't need to Update().
	if equality.Semantic.DeepEqual(iampolicy, newI) {
		glog.V(1).Infof("[IAMPolicy %v] nothing to update", newI.Name)
		return nil
	}
	newI.Status.SyncDetails.Time = metav1.Now()
	if err := r.Update(ctx, newI); err != nil {
		return errors.Wrapf(err, "failed to update SyncDetails of IAMPolicy %s in API server."+
			" Wanted to populate Token %s.", iampolicy.Name, newI.Status.SyncDetails.Token)
	}
	return nil
}
