// Package constraint includes a controller and reconciler for PolicyController constraints.
package constraint

import (
	"context"

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
	controllerPrefix = "policycontroller-constraint-controller-"
	constraintsGroup = "constraints.gatekeeper.sh"
)

var constraintGV = schema.GroupVersion{
	Group:   constraintsGroup,
	Version: "v1beta1",
}

// GVK returns the fully qualified GroupVersionKind of the given constraint kind.
func GVK(kind string) schema.GroupVersionKind {
	return constraintGV.WithKind(kind)
}

// MatchesGroup returns true if the given CRD seems to be in the constraints group.
func MatchesGroup(crd *v1beta1.CustomResourceDefinition) bool {
	return crd.Spec.Group == constraintsGroup
}

// AddController adds a controller for the specified constraint kind to the given Manager.
func AddController(ctx context.Context, mgr manager.Manager, kind string) error {
	gvk := GVK(kind)
	r := newReconciler(ctx, mgr.GetClient(), gvk)

	glog.Infof("Adding controller for constraint: %s", gvk)
	controllerName := controllerPrefix + gvk.String()
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		glog.Errorf("Error creating %s: %v", controllerName, err)
		return err
	}

	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)
	err = c.Watch(&source.Kind{Type: &resource}, &handler.EnqueueRequestForObject{})
	if err != nil {
		glog.Errorf("Error setting up watch for %s: %v", gvk.String(), err)
		return err
	}

	return nil
}
