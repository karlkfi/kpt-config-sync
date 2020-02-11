package constrainttemplate

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/policycontroller/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	annotateConstraintTemplate(ct)
	return reconcile.Result{}, nil
}

// The following structs allow the code to deserialize Gatekeeper types without
// importing them as a direct dependency.
// These types are from templates.gatekeeper.sh/v1beta1

type constraintTemplateStatus struct {
	Created bool          `json:"created,omitempty"`
	ByPod   []byPodStatus `json:"byPod,omitempty"`
}

type byPodStatus struct {
	ID                 string                       `json:"id,omitempty"`
	ObservedGeneration int64                        `json:"observedGeneration,omitempty"`
	Errors             []util.PolicyControllerError `json:"errors,omitempty"`
}

func unmarshalCT(ct unstructured.Unstructured) (*constraintTemplateStatus, error) {
	status := &constraintTemplateStatus{}
	if err := util.UnmarshalStatus(ct, status); err != nil {
		return nil, err
	}
	return status, nil
}

// end Gatekeeper types

// annotateConstraintTemplate processes the given ConstraintTemplate and sets
// Nomos resource status annotations for it.
func annotateConstraintTemplate(ct unstructured.Unstructured) {
	util.ResetAnnotations(&ct)
	gen := ct.GetGeneration()

	status, err := unmarshalCT(ct)
	if err != nil {
		glog.Errorf("Failed to unmarshal ConstraintTemplate %q: %v", ct.GetName(), err)
		return
	}

	if status == nil || !status.Created {
		util.AnnotateUnready(&ct, "ConstraintTemplate has not been created")
		return
	}

	if len(status.ByPod) == 0 {
		util.AnnotateUnready(&ct, "ConstraintTemplate has not been processed by PolicyController")
		return
	}

	var unreadyMsgs []string
	var errorMsgs []string
	for _, bps := range status.ByPod {
		if bps.ObservedGeneration != gen {
			unreadyMsgs = append(unreadyMsgs, fmt.Sprintf("[%s] PolicyController has an outdated version of ConstraintTemplate", bps.ID))
		} else {
			// We only look for errors if the version is up-to-date.
			statusErrs := util.FormatErrors(bps.ID, bps.Errors)
			errorMsgs = append(errorMsgs, statusErrs...)
		}
	}

	if len(unreadyMsgs) > 0 {
		util.AnnotateUnready(&ct, unreadyMsgs...)
	}
	if len(errorMsgs) > 0 {
		util.AnnotateErrors(&ct, errorMsgs...)
	}
}
