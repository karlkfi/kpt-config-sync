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
	"github.com/google/nomos/pkg/bespin-controllers/util"
	"github.com/pkg/errors"
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

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileProject{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("project-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Project
	err = c.Watch(&source.Kind{Type: &bespinv1.Project{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		glog.Errorf("project controller failed to watch instance: %v", err)
		return errors.Wrap(err, "project controller error in adding instance to watch")
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileProject{}

// ReconcileProject reconciles a Project object
type ReconcileProject struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Project object and makes changes based on the state read
// and what is in the Project.Spec
// The comment line below(starting with +kubebuilder) does not work without kubebuilder code layout. It was
// created by kubebuilder in some other repo. Kubebuilder can parse it to generate rbac yaml.
// +kubebuilder:rbac:groups=bespin.dev,resources=projects,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileProject) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Project instance
	instance := &bespinv1.Project{}
	ctx, cancel := context.WithTimeout(context.TODO(), reconcileTimeout)
	defer cancel()
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		glog.Errorf("project reconciler failed to get instance: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "project reconciler error in getting project instance")
	}

	tfs, err := instance.Spec.TFString()
	if err != nil {
		glog.Errorf("project controller failed to setup terraform config: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "project reconciler error in getting terraform config")
	}
	err = util.RunTerraform(tfs)
	if err != nil {
		glog.Errorf("project reconciler failed to run terraform: %v", err)
		return reconcile.Result{}, errors.Wrap(err, "project reconciler error in running terraform")
	}
	return reconcile.Result{}, nil
}
