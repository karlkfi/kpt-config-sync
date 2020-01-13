package constrainttemplate

import (
	"context"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type constraintTemplateReconciler struct {
	ctx    context.Context
	client client.Client
}

func newReconciler(ctx context.Context, cl client.Client) *constraintTemplateReconciler {
	return &constraintTemplateReconciler{ctx, cl}
}

// Reconcile handles Requests from the ConstraintTemplate controller. It will
// annotate ConstraintTemplates based upon their status.
func (c *constraintTemplateReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ct := emptyConstraintTemplate()
	if err := c.client.Get(c.ctx, request.NamespacedName, &ct); err != nil {
		if !errors.IsNotFound(err) {
			glog.Errorf("Error getting ConstraintTemplate %q: %v", request.NamespacedName, err)
			return reconcile.Result{}, err
		}

		glog.Infof("ConstraintTemplate %q was deleted", request.NamespacedName)
		return reconcile.Result{}, nil
	}

	glog.Infof("ConstraintTemplate %q was upserted: %v", request.NamespacedName, ct.Object)
	return reconcile.Result{}, nil
}
