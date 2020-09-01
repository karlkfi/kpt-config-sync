package parse

import (
	"encoding/json"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
)

// gitContext contains the fields which identify where a resource is being synced
// from.
type gitContext struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Rev    string `json:"rev"`
}

func addAnnotationsAndLabels(objs []core.Object, scope declared.Scope, gc gitContext, commitHash string) error {
	gcVal, err := json.Marshal(gc)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		// TODO(b/165798652): Set the declared config for the key v1.DeclaredConfigAnnotationKey
		// here once the new apply logic as required in b/165798652 is implemented.
		core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
		core.SetAnnotation(obj, v1alpha1.GitContextKey, string(gcVal))
		core.SetAnnotation(obj, v1alpha1.ResourceManagerKey, string(scope))
		core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, commitHash)

		value := core.GetAnnotation(obj, v1.ResourceManagementKey)
		if value != v1.ResourceManagementDisabled {
			core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
		}
	}
	return nil
}
