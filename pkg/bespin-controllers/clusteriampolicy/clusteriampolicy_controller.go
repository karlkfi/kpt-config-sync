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

package clusteriampolicy

import (
	"context"
	"time"

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/resource"
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TODO(b/122955229): consolidate the common code in a single place.
const reconcileTimeout = time.Minute * 5

// Add creates a new ClusterIAMPolicy Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager, ef terraform.ExecutorCreator) error {
	return add(mgr, newReconciler(mgr, ef))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, ef terraform.ExecutorCreator) reconcile.Reconciler {
	return &ReconcileClusterIAMPolicy{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		ef:     ef,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller.
	c, err := controller.New("clusteriampolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrap(err, "failed to create new ClusterIAMPolicy controller")
	}

	// Watch for changes to ClusterIAMPolicy.
	err = c.Watch(&source.Kind{Type: &bespinv1.ClusterIAMPolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return errors.Wrap(err, "failed to watch ClusterIAMPolicy")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileClusterIAMPolicy{}

// ReconcileClusterIAMPolicy reconciles a ClusterIAMPolicy object.
type ReconcileClusterIAMPolicy struct {
	client.Client
	scheme *runtime.Scheme
	ef     terraform.ExecutorCreator
}

// Reconcile reads that state of the cluster for a ClusterIAMPolicy object and makes changes based on the state read
// and what is in the ClusterIAMPolicy.Spec.
func (r *ReconcileClusterIAMPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	ciam := &bespinv1.ClusterIAMPolicy{}
	name := request.NamespacedName
	if err := r.Get(ctx, name, ciam); err != nil {
		// Instance was just deleted.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("ClusterIAMPolicy reconciler error in getting Clusteriampolicy instance: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "ClusterIAMPolicy reconciler error in getting clusteriampolicy instance")
	}
	newciam := &bespinv1.ClusterIAMPolicy{}
	ciam.DeepCopyInto(newciam)
	if err := r.setOwnerReference(ctx, newciam); err != nil {
		glog.Errorf("ClusterIAMPolicy reconciler failed to set owner reference: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "ClusterIAMPolicy reconciler failed to set owner reference")
	}
	tfe, err := r.ef.NewExecutor(ctx, r.Client, newciam)
	if err != nil {
		glog.Errorf("ClusterIAMPolicy reconciler failed to create new Terraform executor: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "ClusterIAMPolicy reconciler failed to create new Terraform executor")
	}
	defer func() {
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[ClusterIAMPolicy %v] reconciler failed to close Terraform executor: %v", name, cErr)
		}
	}()

	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		glog.Errorf("ClusterIAMPolicy reconciler failed to run Terraform command: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "ClusterIAMPolicy reconciler failed to run Terraform command")
	}
	if err = resource.Update(ctx, r.Client, ciam, newciam); err != nil {
		err = errors.Wrap(err, "reconciler failed to update IAMPolicy in API server")
		return reconcile.Result{}, err
	}
	glog.V(1).Infof("[ClusterIAMPolicy %v] reconciler successfully finished", name)
	return reconcile.Result{}, nil
}

func (r *ReconcileClusterIAMPolicy) setOwnerReference(ctx context.Context, cip *bespinv1.ClusterIAMPolicy) error {
	refKind := cip.Spec.ResourceRef.Kind
	switch refKind {
	case bespinv1.OrganizationKind, bespinv1.FolderKind:
		owner, err := resource.Get(ctx, r.Client, refKind, cip.Spec.ResourceRef.Name, cip.Namespace)
		if err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(owner, cip, r.scheme); err != nil {
			return errors.Errorf("reconciler failed to set controller reference: %v", err)
		}
		glog.V(1).Infof("[ClusterIAMPolicy %v] successfully set OwnerReference: %v", cip.Name, owner)
	default:
		return errors.Errorf("invalid resource reference reference kind: %v", refKind)
	}
	return nil
}
