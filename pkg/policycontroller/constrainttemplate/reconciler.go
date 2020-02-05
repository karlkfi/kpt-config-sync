package constrainttemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
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
	ID                 string           `json:"id,omitempty"`
	ObservedGeneration int64            `json:"observedGeneration,omitempty"`
	Errors             []createCRDError `json:"errors,omitempty"`
}

type createCRDError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

func unmarshalCT(ct unstructured.Unstructured) (*constraintTemplateStatus, error) {
	statusRaw, found, err := unstructured.NestedFieldNoCopy(ct.Object, "status")
	if err != nil {
		return nil, err
	}
	if !found || statusRaw == nil {
		return nil, nil
	}
	statusJSON, err := json.Marshal(statusRaw)
	if err != nil {
		return nil, err
	}
	status := &constraintTemplateStatus{}
	if err := json.Unmarshal(statusJSON, status); err != nil {
		return nil, err
	}
	return status, nil
}

// end Gatekeeper types

// annotateConstraintTemplate processes the given ConstraintTemplate and sets
// Nomos resource status annotations for it.
func annotateConstraintTemplate(ct unstructured.Unstructured) {
	core.RemoveAnnotations(&ct, nomosv1.ResourceStatusUnreadyKey)
	core.RemoveAnnotations(&ct, nomosv1.ResourceStatusErrorsKey)
	gen := ct.GetGeneration()

	status, err := unmarshalCT(ct)
	if err != nil {
		glog.Errorf("Failed to unmarshal ConstraintTemplate %q: %v", ct.GetName(), err)
		return
	}

	if !status.Created {
		core.SetAnnotation(&ct, nomosv1.ResourceStatusUnreadyKey, "ConstraintTemplate has not been created")
		return
	}

	if len(status.ByPod) == 0 {
		core.SetAnnotation(&ct, nomosv1.ResourceStatusUnreadyKey, "ConstraintTemplate has not been processed by PolicyController")
		return
	}

	var unreadyMsgs []string
	var errorMsgs []string
	for _, bps := range status.ByPod {
		if bps.ObservedGeneration != gen {
			unreadyMsgs = append(unreadyMsgs, fmt.Sprintf("[%s] PolicyController has an outdated version of ConstraintTemplate", bps.ID))
		} else {
			// We only look for errors if the version is up-to-date.
			statusErrs := statusErrors(bps)
			errorMsgs = append(errorMsgs, statusErrs...)
		}
	}

	if len(unreadyMsgs) > 0 {
		core.SetAnnotation(&ct, nomosv1.ResourceStatusUnreadyKey, jsonify(unreadyMsgs))
	}
	if len(errorMsgs) > 0 {
		core.SetAnnotation(&ct, nomosv1.ResourceStatusErrorsKey, jsonify(errorMsgs))
	}
}

// statusErrors flattens the errors field of the given byPodStatus into a string
// array.
func statusErrors(bps byPodStatus) []string {
	var errs []string
	for _, cce := range bps.Errors {
		var prefix string
		if len(cce.Code) > 0 {
			prefix = fmt.Sprintf("[%s] %s:", bps.ID, cce.Code)
		} else {
			prefix = fmt.Sprintf("[%s]:", bps.ID)
		}

		msg := cce.Message
		if len(msg) == 0 {
			msg = "[missing PolicyController error]"
		}

		errs = append(errs, fmt.Sprintf("%s %s", prefix, msg))
	}
	return errs
}

// jsonify marshals the given string array as a JSON string.
func jsonify(strs []string) string {
	errJSON, err := json.Marshal(strs)
	if err == nil {
		return string(errJSON)
	}

	// This code is not intended to be reached. It just provides a sane fallback
	// if there is ever an error from json.Marshal().
	glog.Errorf("Failed to JSONify strings: %v", err)
	var b strings.Builder
	b.WriteString("[")
	for i, statusErr := range strs {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("%q", statusErr))
	}
	b.WriteString("]")
	return b.String()
}
