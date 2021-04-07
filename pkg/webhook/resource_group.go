package webhook

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// isResourceGroupRequest returns true if the request is for a ResourceGroup CR.
func isResourceGroupRequest(req admission.Request) bool {
	gk := schema.GroupKind{
		Group: req.Kind.Group,
		Kind:  req.Kind.Kind,
	}
	return gk == live.ResourceGroupGVK.GroupKind()
}

// handleResourceGroupRequest handles the request with following rules:
// If the ResourceGroup CR is generated by ConfigSync, users can't modify it.
// If the ResourceGroup CR is not generated by ConfigSync, users can modify it.
func handleResourceGroupRequest(req admission.Request) admission.Response {
	fromConfigSync, err := fromConfigSync(req)
	if err != nil {
		glog.Errorf("Unable to read labels from new object: %v", req.Object.Object)
		return allow()
	}
	if fromConfigSync {
		return deny(metav1.StatusReasonUnauthorized, fmt.Sprintf("requester is not authorized to modify %s generated by Config Sync", live.ResourceGroupGVK.GroupKind()))
	}
	// Allow ResourceGroups that are not generated by ConfigSync.
	return allow()
}

var metadataAccessor = meta.NewAccessor()

// fromConfigSync returns true if the ResourceGroup is generated by ConfigSync.
func fromConfigSync(req admission.Request) (bool, error) {
	name := req.Name
	namespace := req.Namespace

	// Favor the original version of the object (unless this is a CREATE in which
	// case there is only the new version).
	obj := req.OldObject.Object
	if obj == nil {
		obj = req.Object.Object
	}

	labels, err := metadataAccessor.Labels(obj)
	if err != nil {
		return false, err
	}

	hasInventoryLabel := labels[common.InventoryLabel] == applier.InventoryID(namespace)

	if namespace == configsync.ControllerNamespace {
		return name == v1alpha1.RootSyncName && hasInventoryLabel, nil
	}
	return name == v1alpha1.RepoSyncName && hasInventoryLabel, nil
}
