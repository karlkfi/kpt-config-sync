package webhook

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AddValidator adds the admission webhook validator to the passed manager.
func AddValidator(mgr manager.Manager) error {
	handler, err := handler(mgr.GetConfig())
	if err != nil {
		return err
	}
	mgr.GetWebhookServer().Register(configuration.ServingPath, &webhook.Admission{
		Handler: handler,
	})
	return nil
}

// Validator is the part of the validating webhook which handles admission
// requests and admits or denies them.
type Validator struct {
	differ *ObjectDiffer
}

var _ admission.Handler = &Validator{}

// Handler returns a Validator which satisfies the admission.Handler interface.
func handler(cfg *rest.Config) (*Validator, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	vc, err := declared.NewValueConverter(dc)
	if err != nil {
		return nil, err
	}
	return &Validator{&ObjectDiffer{vc}}, nil
}

// Handle implements admission.Handler
func (v *Validator) Handle(_ context.Context, req admission.Request) admission.Response {
	// An admission request for a sub-resource (such as a Scale) will not include
	// the full parent for us to validate until the admission chain is fixed:
	// https://github.com/kubernetes/enhancements/pull/1600
	// Until then, we will not configure the webhook to intercept subresources so
	// this block should never be reached.
	if req.SubResource != "" {
		glog.Errorf("Unable to review admission request for sub-resource: %v", req)
		return allow()
	}

	// Check UserInfo for Config Sync service account and handle if found.
	if isConfigSyncSA(req.UserInfo) {
		username := req.UserInfo.Username
		if isImporter(username) {
			// Config Sync importer can always update a resource.
			return allow()
		}
		// Perform manager precedence check to verify this Config Sync reconciler
		// can manage the object.
		mgr, err := objectManager(req)
		if err != nil {
			glog.Error(err.Error())
			return allow()
		}
		if canManage(username, mgr) {
			return allow()
		}
		return deny(metav1.StatusReasonUnauthorized, fmt.Sprintf("%s can not manage object which is already managed by %s", username, mgr))
	}

	// Handle the requests for ResourceGroup CRs.
	if isResourceGroupRequest(req) {
		return handleResourceGroupRequest(req)
	}

	// Convert to client.Objects for convenience.
	oldObj, newObj, err := convertObjects(req)
	if err != nil {
		glog.Error(err.Error())
		return allow()
	}

	switch req.Operation {
	case admissionv1.Create:
		return v.handleCreate(newObj)
	case admissionv1.Delete:
		return v.handleDelete(oldObj)
	case admissionv1.Update:
		return v.handleUpdate(oldObj, newObj)
	default:
		glog.Errorf("Unsupported operation: %v", req.Operation)
		return allow()
	}
}

func (v *Validator) handleCreate(newObj client.Object) admission.Response {
	if configSyncManaged(newObj) {
		return deny(metav1.StatusReasonUnauthorized, "requester is not authorized to create managed resources")
	}
	return allow()
}

func (v *Validator) handleDelete(oldObj client.Object) admission.Response {
	if configSyncManaged(oldObj) {
		return deny(metav1.StatusReasonUnauthorized, "requester is not authorized to delete managed resources")
	}
	return allow()
}

func (v *Validator) handleUpdate(oldObj, newObj client.Object) admission.Response {
	// Verify that old and/or new objects are managed by Config Sync.
	if !configSyncManaged(oldObj, newObj) {
		// The webhook should be configured to only intercept resources which are
		// managed by Config Sync.
		glog.Warningf("Received admission request for unmanaged object: %v", newObj)
		return allow()
	}

	// Build a diff set between old and new objects.
	diffSet, err := v.differ.FieldDiff(oldObj, newObj)
	if err != nil {
		glog.Errorf("Failed to generate field diff set: %v", err)
		return allow()
	}

	// If the diff set includes any ConfigSync labels or annotations, reject the
	// request immediately.
	if csSet := ConfigSyncMetadata(diffSet); !csSet.Empty() {
		return deny(metav1.StatusReasonForbidden, "Config Sync metadata can not be modified: "+csSet.String())
	}

	// Use the ConfigSync declared fields annotation to build the set of fields
	// which should not be modified.
	declaredSet, err := DeclaredFields(oldObj)
	if err != nil {
		glog.Errorf("Failed to decoded declared fields: %v", err)
		return allow()
	}

	// If the diff set and declared set have any fields in common, reject the
	// request. Otherwise allow it.
	invalidSet := diffSet.Intersection(declaredSet)
	if !invalidSet.Empty() {
		return deny(metav1.StatusReasonForbidden, "fields managed by Config Sync can not be modified: "+invalidSet.String())
	}
	return allow()
}

func convertObjects(req admission.Request) (client.Object, client.Object, error) {
	var oldObj client.Object
	switch {
	case req.OldObject.Object != nil:
		// We got an already-parsed object.
		var ok bool
		oldObj, ok = req.OldObject.Object.(client.Object)
		if !ok {
			return nil, nil, fmt.Errorf("failed to convert to client.Object: %v", req.OldObject.Object)
		}
	case req.OldObject.Raw != nil:
		// We got raw JSON bytes instead of an object.
		oldU := &unstructured.Unstructured{}
		if err := oldU.UnmarshalJSON(req.OldObject.Raw); err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert to client.Object: %v", req.OldObject.Raw)
		}
		oldObj = oldU
	}

	var newObj client.Object
	switch {
	case req.Object.Object != nil:
		// We got an already-parsed object.
		var ok bool
		newObj, ok = req.Object.Object.(client.Object)
		if !ok {
			return nil, nil, fmt.Errorf("failed to convert to client.Object: %v", req.Object.Object)
		}
	case req.Object.Raw != nil:
		// We got raw JSON bytes instead of an object.
		newU := &unstructured.Unstructured{}
		if err := newU.UnmarshalJSON(req.Object.Raw); err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert to client.Object: %v", req.Object.Raw)
		}
		newObj = newU
	}
	return oldObj, newObj, nil
}

func configSyncManaged(objs ...client.Object) bool {
	for _, obj := range objs {
		if obj != nil && obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementEnabled {
			return true
		}
	}
	return false
}

func objectManager(req admission.Request) (string, error) {
	oldObj, newObj, err := convertObjects(req)
	if err != nil {
		return "", err
	}
	mgr := getManager(oldObj)
	if mgr == "" {
		mgr = getManager(newObj)
	}
	return mgr, nil
}

func getManager(obj client.Object) string {
	if obj == nil {
		return ""
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return ""
	}
	return annotations[v1alpha1.ResourceManagerKey]
}

func allow() admission.Response {
	return admission.Allowed("")
}

func deny(reason metav1.StatusReason, message string) admission.Response {
	resp := admission.Denied(string(reason))
	resp.Result.Message = message
	return resp
}
