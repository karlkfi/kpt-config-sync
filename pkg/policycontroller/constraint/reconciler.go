package constraint

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/policycontroller/util"
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

	glog.Infof("%s %q was upserted", c.gvk, request.NamespacedName)
	patch := client.MergeFrom(resource.DeepCopy())
	annotateConstraint(resource)
	err := c.client.Patch(c.ctx, &resource, patch)
	if err != nil {
		glog.Errorf("Failed to patch annotations for %s: %v", c.gvk, err)
	}
	return reconcile.Result{}, err
}

// The following structs allow the code to deserialize Gatekeeper constraints.
// Note that these constraints are generated dynamically so there is no schema
// to import or link against.
// The structs align with constraints.gatekeeper.sh/v1beta1

type constraintStatus struct {
	ByPod []byPodStatus `json:"byPod,omitempty"`
}

type byPodStatus struct {
	ID                 string                       `json:"id,omitempty"`
	ObservedGeneration int64                        `json:"observedGeneration,omitempty"`
	Errors             []util.PolicyControllerError `json:"errors,omitempty"`
	Enforced           bool                         `json:"enforced,omitempty"`
}

func unmarshalConstraint(ct unstructured.Unstructured) (*constraintStatus, error) {
	status := &constraintStatus{}
	if err := util.UnmarshalStatus(ct, status); err != nil {
		return nil, err
	}
	return status, nil
}

// end Gatekeeper types

// annotateConstraint processes the given Constraint and sets Nomos resource
// status annotations for it.
func annotateConstraint(con unstructured.Unstructured) {
	util.ResetAnnotations(&con)
	gen := con.GetGeneration()

	status, err := unmarshalConstraint(con)
	if err != nil {
		glog.Errorf("Failed to unmarshal %s %q: %v", con.GroupVersionKind(), con.GetName(), err)
		return
	}

	if status == nil || len(status.ByPod) == 0 {
		util.AnnotateReconciling(&con, "Constraint has not been processed by PolicyController")
		return
	}

	var reconcilingMsgs []string
	var errorMsgs []string
	for _, bps := range status.ByPod {
		// We only look for errors/enforcement if the version is up-to-date.
		if bps.ObservedGeneration != gen {
			reconcilingMsgs = append(reconcilingMsgs, fmt.Sprintf("[%s] PolicyController has an outdated version of Constraint", bps.ID))
			continue
		}

		if !bps.Enforced {
			reconcilingMsgs = append(reconcilingMsgs, fmt.Sprintf("[%s] PolicyController is not enforcing Constraint", bps.ID))
		}
		statusErrs := util.FormatErrors(bps.ID, bps.Errors)
		errorMsgs = append(errorMsgs, statusErrs...)
	}

	if len(reconcilingMsgs) > 0 {
		util.AnnotateReconciling(&con, reconcilingMsgs...)
	}
	if len(errorMsgs) > 0 {
		util.AnnotateErrors(&con, errorMsgs...)
	}
}
