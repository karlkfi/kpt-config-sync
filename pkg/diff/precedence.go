// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diff

import (
	"fmt"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/klog/v2"
	"kpt.dev/configsync/pkg/core"
	"kpt.dev/configsync/pkg/declared"
	"kpt.dev/configsync/pkg/importer"
	"kpt.dev/configsync/pkg/kinds"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/reconcilermanager"
	"kpt.dev/configsync/pkg/syncer/differ"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	saImporter             = importer.Name
	saRootReconcilerPrefix = core.RootReconcilerPrefix
	saReconcilerManager    = reconcilermanager.ManagerName

	// OperationManage is a meta operation that implies full control:
	// CREATE + UPDATE + DELETE
	OperationManage = admissionv1.Operation("MANAGE")
)

// IsManager returns true if the given reconciler is the manager for the resource.
func IsManager(scope declared.Scope, syncName string, obj client.Object) bool {
	if !differ.ManagedByConfigSync(obj) {
		// Not managed
		return false
	}
	newManager := declared.ResourceManager(scope, syncName)
	oldManager := core.GetAnnotation(obj, metadata.ResourceManagerKey)
	return oldManager == newManager
}

// CanManage returns true if the given reconciler is allowed to perform the
// specified operation on the specified resource object.
func CanManage(scope declared.Scope, syncName string, obj client.Object, op admissionv1.Operation) bool {
	if !differ.ManagementEnabled(obj) {
		// Not managed
		return true
	}
	oldManager := core.GetAnnotation(obj, metadata.ResourceManagerKey)
	newManager := declared.ResourceManager(scope, syncName)
	reconciler, _ := reconcilerName(newManager)
	err := ValidateManager(reconciler, oldManager, core.IDOf(obj), op)
	if err != nil {
		klog.V(3).Infof("diff.CanManage? %v", err)
		return false
	}
	return true
}

// ValidateManager returns nil if the given reconciler is allowed to perform the
// specified operation on the resource object with the specified id.
//
// It's not possible to parse a ReconcilerName into its component parts, because
// R*Sync names and namespaces may include a dash, which is used as the delimiter.
// But you CAN parse a manager name into scope (namespace) and name, and use that
// to build a reconciler name. So we have to compare reconciler names instead of
// manager names.
func ValidateManager(reconciler, manager string, id core.ID, op admissionv1.Operation) error {
	if manager == "" {
		// All managers are allowed to manage an object without a specified manager
		return nil
	}

	// TODO(b/161167923): Remove this check when we turn down the old importer deployment.
	if isImporter(reconciler) {
		// Config Sync importer (legacy) is allowed to manage any object.
		return nil
	}

	if isReconcilerManager(reconciler) && op == admissionv1.Update &&
		(id.GroupKind == kinds.RootSyncV1Beta1().GroupKind() ||
			id.GroupKind == kinds.RepoSyncV1Beta1().GroupKind()) {
		// ReconcilerManager is allowed to update RootSync/RepoSync to add/remove the finalizer
		return nil
	}

	oldReconciler, syncScope := reconcilerName(manager)

	if err := declared.ValidateScope(string(syncScope)); err != nil {
		// All managers are allowed to manage an object with an invalid manager.
		// But print a warning, because users shouldn't manually modify the manager.
		klog.Warningf("Invalid manager annotation (object: %q, annotation: %s=%q)", id, metadata.ResourceManagerKey, manager)
	}

	if isRootReconciler(reconciler) && syncScope != declared.RootReconciler {
		// RootReconciler is allowed to adopt an object as long as it's not
		// managed by another RootReconciler.
		return nil
	}

	if reconciler != oldReconciler {
		return fmt.Errorf("config sync %q can not %s object %q managed by config sync %q",
			reconciler, op, id, oldReconciler)
	}
	return nil
}

func reconcilerName(manager string) (string, declared.Scope) {
	syncScope, syncName := declared.ManagerScopeAndName(manager)
	var reconciler string
	if syncScope == declared.RootReconciler {
		reconciler = core.RootReconcilerName(syncName)
	} else {
		reconciler = core.NsReconcilerName(string(syncScope), syncName)
	}
	return reconciler, syncScope
}

func isImporter(reconcilerName string) bool {
	return reconcilerName == saImporter
}

func isRootReconciler(reconcilerName string) bool {
	return strings.HasPrefix(reconcilerName, saRootReconcilerPrefix)
}

func isReconcilerManager(reconcilerName string) bool {
	return reconcilerName == saReconcilerManager
}
