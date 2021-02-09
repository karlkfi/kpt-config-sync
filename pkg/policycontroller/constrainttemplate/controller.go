// Package constrainttemplate includes a controller and reconciler for PolicyController constraint templates.
package constrainttemplate

import (
	"github.com/golang/glog"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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
	gk = schema.GroupKind{
		Group: templatesGroup,
		Kind:  "ConstraintTemplate",
	}

	// GVK is the GVK for gatekeeper ConstraintTemplates.
	GVK = gk.WithVersion("v1beta1")
)

// MatchesGK returns true if the given CRD defines the gatekeeper ConstraintTemplate.
func MatchesGK(crd *v1beta1.CustomResourceDefinition) bool {
	return crd.Spec.Group == templatesGroup && crd.Spec.Names.Kind == gk.Kind
}

// AddController adds the ConstraintTemplate controller to the given Manager.
func AddController(mgr manager.Manager) error {
	glog.Info("Adding controller for ConstraintTemplates")
	r := newReconciler(mgr.GetClient())
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	ct := emptyConstraintTemplate()
	return c.Watch(&source.Kind{Type: &ct}, &handler.EnqueueRequestForObject{})
}

func emptyConstraintTemplate() unstructured.Unstructured {
	ct := unstructured.Unstructured{}
	ct.SetGroupVersionKind(GVK)
	return ct
}
