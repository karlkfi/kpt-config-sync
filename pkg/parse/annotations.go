package parse

import (
	"encoding/json"
	"fmt"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
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
		core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
		core.SetAnnotation(obj, constants.GitContextKey, string(gcVal))
		core.SetAnnotation(obj, constants.ResourceManagerKey, string(scope))
		core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, commitHash)
		core.SetAnnotation(obj, constants.ResourceIDKey, core.GKNN(obj))
		core.SetAnnotation(obj, constants.OwningInventoryKey, inventoryID)

		value := core.GetAnnotation(obj, v1.ResourceManagementKey)
		if value != v1.ResourceManagementDisabled {
			core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
		}
	}
	return nil
}
