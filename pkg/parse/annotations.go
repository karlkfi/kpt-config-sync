package parse

import (
	"encoding/json"
	"fmt"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/metadata"
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
		return fmt.Errorf("marshaling gitContext: %w", err)
	}
	var inventoryID string
	if scope == declared.RootReconciler {
		inventoryID = applier.InventoryID(configmanagement.ControllerNamespace)
	} else {
		inventoryID = applier.InventoryID(string(scope))
	}
	for _, obj := range objs {
		core.SetLabel(obj, metadata.ManagedByKey, metadata.ManagedByValue)
		core.SetAnnotation(obj, metadata.GitContextKey, string(gcVal))
		core.SetAnnotation(obj, metadata.ResourceManagerKey, string(scope))
		core.SetAnnotation(obj, metadata.SyncTokenAnnotationKey, commitHash)
		core.SetAnnotation(obj, metadata.ResourceIDKey, core.GKNN(obj))
		core.SetAnnotation(obj, metadata.OwningInventoryKey, inventoryID)

		value := core.GetAnnotation(obj, metadata.ResourceManagementKey)
		if value != metadata.ResourceManagementDisabled {
			core.SetAnnotation(obj, metadata.ResourceManagementKey, metadata.ResourceManagementEnabled)
		}
	}
	return nil
}
