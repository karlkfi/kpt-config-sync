package webhook

import (
	"context"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// validator is the part of the validating webhook which handles admission
// requests and admits or denies them.
type validator struct{}

var _ admission.Handler = &validator{}

// Handle implements admission.Handler
func (v *validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if reason := skipValidation(req); reason != "" {
		// This admission controller is part of a defense-in-depth strategy. Config
		// Sync is still running a reconciler which will revert any unwanted changes
		// that get through. If the request does not meet all the preconditions for
		// it to be validated, we will fail open and allow the reconciler to revert
		// any changes as necessary.
		return admission.Allowed(reason)
	}

	// Check UserInfo for ConfigSync and perform manager precedence check.
	if isConfigSyncSA(req.UserInfo) {
		if isImporter(req.UserInfo.Username) {
			return admission.Allowed("Config Sync importer can always update a resource")
		}
		// TODO(b/160786209): Add manager precedence checks once service accounts are known for root and repo reconcilers.
	}

	// TODO(b/160786928): Build diff list between old and new objects

	// TODO(b/160786679): If the diff list includes any ConfigSync labels or annotations, reject the request immediately.

	// TODO(b/160786679): Use the ConfigSync managed fields annotation to build an “immutable list” of which fields should not be modified.
	// TODO(b/160786679): Handle the case where management is being enabled or disabled

	// TODO(b/160786679): If the diff list and immutable list have any fields in common, reject the request. Otherwise allow it.

	return admission.Allowed("")
}

// skipValidation checks to see if the given Request meets all preconditions for
// validation. If the Request fails to meet any, this returns a string
// describing why the Request should skip validation.
func skipValidation(req admission.Request) string {
	// An admission request for a sub-resource (such as a Scale) will not include
	// the full parent for us to validate until the admission chain is fixed:
	// https://github.com/kubernetes/enhancements/pull/1600
	// Until then, we will not configure the webhook to intercept subresources so
	// this block should never be reached.
	if req.SubResource != "" {
		glog.Errorf("Unable to review admission request for sub-resource: %v", req)
		return "unable to review admission request for sub-resource"
	}

	// Verify that old and/or new objects are managed by Config Sync.
	oldMng, err := isManaged(req.OldObject.Object)
	if err != nil {
		glog.Errorf("Unable to read annotations from old object: %v", req.OldObject.Object)
		return "unable to read annotations from old object"
	}
	mng, err := isManaged(req.Object.Object)
	if err != nil {
		glog.Errorf("Unable to read annotations from new object: %v", req.Object.Object)
		return "unable to read annotations from new object"
	}
	// The webhook should be configured to only intercept resources which are
	// managed by Config Sync.
	if !oldMng && !mng {
		glog.Warningf("Received admission request for unmanaged object: %v", req.OldObject.Object)
		return "object is not managed by Config Sync"
	}

	return ""
}

var metadataAccessor = meta.NewAccessor()

func isManaged(obj runtime.Object) (bool, error) {
	annots, err := metadataAccessor.Annotations(obj)
	if err != nil {
		return false, err
	}
	return annots[v1.ResourceManagementKey] == v1.ResourceManagementEnabled, nil
}
