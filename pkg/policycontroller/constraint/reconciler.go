package constraint

import (
	"context"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type constraintReconciler struct {
	ctx    context.Context
	client client.Client
	// gvk is the GroupVersionKind of the constraint resources to reconcile.
	gvk schema.GroupVersionKind
}

func newReconciler(ctx context.Context, client client.Client, gvk schema.GroupVersionKind) *constraintReconciler {
	return &constraintReconciler{ctx, client, gvk}
}

// Reconcile handles Requests from the constraint controller. It will annotate
// Constraints based upon their status.
func (c *constraintReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(c.gvk)
	if err := c.client.Get(c.ctx, request.NamespacedName, &resource); err != nil {
		if errors.IsNotFound(err) {
			glog.Infof("%s %q was deleted", c.gvk, request.NamespacedName)
			return reconcile.Result{}, nil
		}

		glog.Errorf("Error getting %s %q: %v", c.gvk, request.NamespacedName, err)
		return reconcile.Result{}, err
	}

	glog.Infof("%s %q was upserted: %v", c.gvk, request.NamespacedName, resource.Object)
	return reconcile.Result{}, nil
}
