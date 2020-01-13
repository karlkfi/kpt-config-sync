// Package policycontroller includes a meta-controller and reconciler for
// PolicyController resources.
package policycontroller

import (
	"context"

	"github.com/google/nomos/pkg/policycontroller/constrainttemplate"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "policycontroller-controller"
)

// AddControllers sets up the ConstraintTemplate controller as well as a meta
// controller which manages the watches on all Constraints.
func AddControllers(ctx context.Context, mgr manager.Manager) error {
	if err := constrainttemplate.AddController(ctx, mgr); err != nil {
		return err
	}

	r, err := newReconciler(ctx, mgr)
	if err != nil {
		return err
	}

	pc, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// The meta-controller watches CRDs since each new ConstraintTemplate will
	// result in a new CRD for the corresponding type of Constraint.
	return pc.Watch(&source.Kind{Type: &v1beta1.CustomResourceDefinition{}}, &handler.EnqueueRequestForObject{})
}
