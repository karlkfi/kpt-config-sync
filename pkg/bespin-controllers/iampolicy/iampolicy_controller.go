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
	"fmt"
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/resource"
	"github.com/google/nomos/pkg/bespin-controllers/terraform"
	"github.com/google/nomos/pkg/bespin-controllers/test/k8s"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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

const (
	reconcileTimeout = time.Minute * 5
	controllerName   = "iampolicy-controller"
)

// Add creates a new IAMPolicy Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager, ef terraform.ExecutorCreator) error {
	return add(mgr, newReconciler(mgr, ef))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, ef terraform.ExecutorCreator) reconcile.Reconciler {
	return &ReconcileIAMPolicy{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder(controllerName),
		ef:       ef,
	}
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
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	ef       terraform.ExecutorCreator
}

// Reconcile reads that state of the cluster for a IAMPolicy object and makes changes based on the state read
// and what is in the IAMPolicy.Spec.
func (r *ReconcileIAMPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	iam := &bespinv1.IAMPolicy{}
	name := request.NamespacedName
	if err := r.Get(ctx, name, iam); err != nil {
		// Instance was just deleted.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("IAMPolicy reconciler error in getting iampolicy instance: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler error in getting iampolicy instance")
	}
	newiam := &bespinv1.IAMPolicy{}
	iam.DeepCopyInto(newiam)
	if err := r.setOwnerReference(ctx, newiam); err != nil {
		glog.Errorf("IAMPolicy reconciler failed to set owner reference: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler failed to set owner reference")
	}
	tfe, err := r.ef.NewExecutor(ctx, r.Client, newiam)
	if err != nil {
		glog.Errorf("IAMPolicy reconciler failed to create new Terraform executor: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler failed to create new Terraform executor")
	}
	defer func() {
		if cErr := tfe.Close(); cErr != nil {
			glog.Errorf("[IAMPolicy %v] reconciler failed to close Terraform executor: %v", name, cErr)
		}
	}()

	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		glog.Errorf("IAMPolicy reconciler failed to run Terraform command: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "IAMPolicy reconciler failed to run Terraform command")
	}

	if err = resource.Update(ctx, r.Client, iam, newiam); err != nil {
		err = errors.Wrap(err, "reconciler failed to update IAMPolicy in API server")
		return reconcile.Result{}, err
	}

	newCondition := k8s.NewCustomReadyCondition(v1.ConditionTrue, k8s.Updated, k8s.UpdatedMessage)
	if err := r.updateAPIServerInstanceCondition(newiam, newCondition); err != nil {
		return reconcile.Result{}, err
	}
	r.recorder.Eventf(newiam, v1.EventTypeNormal, k8s.Updated, k8s.UpdatedMessage)

	glog.V(1).Infof("[IAMPolicy %v] reconciler successfully finished", name)
	return reconcile.Result{}, nil
}

func (r *ReconcileIAMPolicy) setOwnerReference(ctx context.Context, iampolicy *bespinv1.IAMPolicy) error {
	refKind := iampolicy.Spec.ResourceRef.Kind
	switch refKind {
	case bespinv1.ProjectKind:
		owner, err := resource.Get(ctx, r.Client, refKind, iampolicy.Spec.ResourceRef.Name, iampolicy.Namespace)
		if err != nil {
			return err
		}
		if err := controllerutil.SetControllerReference(owner, iampolicy, r.scheme); err != nil {
			return errors.Errorf("reconciler failed to set controller reference: %v", err)
		}
		glog.V(1).Infof("[IAMPolicy %v] successfully set OwnerReference: %v", iampolicy.Name, owner)
	default:
		return errors.Errorf("invalid resource reference reference kind: %v", refKind)
	}
	return nil
}

func (r *ReconcileIAMPolicy) updateAPIServerInstanceCondition(iam *bespinv1.IAMPolicy,
	newCondition bespinv1.Condition) error {
	iam.Status.Conditions = []bespinv1.Condition{newCondition}
	err := r.Update(context.Background(), iam)
	if err != nil {
		return fmt.Errorf("error updating instance '%v' in api server: %v", r, err)
	}
	return nil
}
