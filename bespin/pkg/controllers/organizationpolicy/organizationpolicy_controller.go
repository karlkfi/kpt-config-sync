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

package organizationpolicy

import (
	"context"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	bespinv1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	"github.com/google/nomos/bespin/pkg/controllers/terraform"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/google/nomos/bespin/pkg/controllers/resource"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/golang/glog"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const controllerName = "iampolicy-controller"

// Add creates a new OrganizationPolicy Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this bespin.Add(mgr) to install this Controller
func Add(mgr manager.Manager, ef terraform.ExecutorCreator) error {
	return add(mgr, newReconciler(mgr, ef))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, ef terraform.ExecutorCreator) reconcile.Reconciler {
	return &ReconcileOrganizationPolicy{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		ef:       ef,
		recorder: mgr.GetRecorder(controllerName)}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("organizationpolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to OrganizationPolicy
	err = c.Watch(&source.Kind{Type: &bespinv1.OrganizationPolicy{}}, &handler.EnqueueRequestForObject{})
	return err
}

var _ reconcile.Reconciler = &ReconcileOrganizationPolicy{}

// ReconcileOrganizationPolicy reconciles a OrganizationPolicy object
type ReconcileOrganizationPolicy struct {
	client.Client
	scheme   *runtime.Scheme
	ef       terraform.ExecutorCreator
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a OrganizationPolicy object and makes changes based on the state read
// and what is in the OrganizationPolicy.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bespin.dev,resources=organizationpolicies,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileOrganizationPolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), resource.ReconcileTimeout)
	defer cancel()
	// Fetch the OrganizationPolicy instance
	instance := &bespinv1.OrganizationPolicy{}
	name := request.NamespacedName
	if err := r.Get(ctx, name, instance); err != nil {
		// Instance was just deleted.
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		glog.Errorf("[OrganizationPolicy %v] failed to get instance: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[OrganizationPolicy %v] failed to get instance", name)
	}
	newInstance := &bespinv1.OrganizationPolicy{}
	instance.DeepCopyInto(newInstance)
	resourceRef := instance.Spec.ResourceRef

	// Fetch the owner so we can set the reference on the controller.
	owner, err := resource.Get(ctx, r.Client, resourceRef.Kind, resourceRef.Name, resourceRef.Namespace)
	if err != nil {
		glog.Errorf("[OrganizationPolicy %v] failed to get reference resource %v", instance, resourceRef)
		return reconcile.Result{}, err
	}
	if err = controllerutil.SetControllerReference(owner, newInstance, r.scheme); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "reconciler failed to set controller reference: %v", instance)
	}
	tfe, err := r.ef.NewExecutor(ctx, r.Client, newInstance)
	if err != nil {
		glog.Errorf("[OrganizationPolicy %v] reconciler failed to create new Terraform executor: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[OrganizationPolicy %v] reconciler failed to create new Terraform executor", name)
	}
	defer func() {
		if cerr := tfe.Close(); err != nil {
			glog.Errorf("[OrganizationPolicy %v] reconciler failed to close Terraform executor: %v", name, cerr)
		}
	}()

	if err = tfe.RunCreateOrUpdateFlow(); err != nil {
		glog.Errorf("[OrganizationPolicy %v] reconciler failed to run Terraform command: %v", name, err)
		return reconcile.Result{}, errors.Wrapf(err,
			"[OrganizationPolicy %v] reconciler failed to run Terraform command", name)
	}
	done, err := resource.Update(ctx, r.Client, r.recorder, instance, newInstance)
	if err != nil {
		err = errors.Wrap(err, "reconciler failed to update instance in API server")
		return reconcile.Result{}, err
	}

	if done {
		glog.V(1).Infof("[OrganizationPolicy %v] reconciler successfully finished", name)
	}

	return reconcile.Result{}, nil
}
