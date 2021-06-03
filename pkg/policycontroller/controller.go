// Package policycontroller includes a meta-controller and reconciler for
// PolicyController resources.
package policycontroller

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "policycontroller-annotator"
)

// AddControllers sets up a meta controller which manages the watches on all
// Constraints and ConstraintTemplates.
func AddControllers(ctx context.Context, mgr manager.Manager) error {
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
	return pc.Watch(&source.Kind{Type: &apiextensionsv1.CustomResourceDefinition{}}, &handler.EnqueueRequestForObject{})
}
