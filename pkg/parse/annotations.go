package parse

import (
	"encoding/json"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kptapplier"
)

// gitContext contains the fields which identify where a resource is being synced
// from.
type gitContext struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Rev    string `json:"rev"`
}

func addAnnotationsAndLabels(objs []ast.FileObject, scope declared.Scope, gc gitContext, commitHash string) error {
	gcVal, err := json.Marshal(gc)
	if err != nil {
		return err
	}
	var inventoryID string
	if scope == declared.RootReconciler {
		inventoryID = kptapplier.InventoryID(configmanagement.ControllerNamespace)
	} else {
		inventoryID = kptapplier.InventoryID(string(scope))
	}
	for _, obj := range objs {
		core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
		core.SetAnnotation(obj, v1alpha1.GitContextKey, string(gcVal))
		core.SetAnnotation(obj, v1alpha1.ResourceManagerKey, string(scope))
		core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, commitHash)

		// set the owning-inventory annotation
		// TODO(b/178744996): Remove setting the owning-inventory once the remediator
		// is able to use kpt live apply library.
		core.SetAnnotation(obj, kptapplier.OwningInventoryKey, inventoryID)

		value := core.GetAnnotation(obj, v1.ResourceManagementKey)
		if value != v1.ResourceManagementDisabled {
			core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
		}
	}
	return nil
}
