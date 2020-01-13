// Package constrainttemplate includes a controller and reconciler for PolicyController constraint templates.
package constrainttemplate

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "policycontroller-constrainttemplate-controller"
	templatesGroup = "templates.gatekeeper.sh"
)

var (
	constraintTemplateGK = schema.GroupKind{
		Group: templatesGroup,
		Kind:  "ConstraintTemplate",
	}

	constraintTemplateGVK = constraintTemplateGK.WithVersion("v1beta1")
)

// AddController adds the ConstraintTemplate controller to the given Manager.
func AddController(ctx context.Context, mgr manager.Manager) error {
	r := newReconciler(ctx, mgr.GetClient())
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	ct := emptyConstraintTemplate()
	return c.Watch(&source.Kind{Type: &ct}, &handler.EnqueueRequestForObject{})
}

func emptyConstraintTemplate() unstructured.Unstructured {
	ct := unstructured.Unstructured{}
	ct.SetGroupVersionKind(constraintTemplateGVK)
	return ct
}
