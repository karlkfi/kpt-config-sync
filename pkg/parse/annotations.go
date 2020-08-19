package parse

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
)

func addAnnotationsAndLabels(objs []core.Object, scope, gitRef, gitRepo, commitHash string) {
	for _, obj := range objs {
		// TODO(b/165798652): Set the declared config for the key v1.DeclaredConfigAnnotationKey
		// here once the new apply logic as required in b/165798652 is implemented.
		core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
		core.SetAnnotation(obj, v1.GitRefKey, gitRef)
		core.SetAnnotation(obj, v1.GitRepoKey, gitRepo)
		core.SetAnnotation(obj, v1.ResourceManagerKey, scope)
		core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, commitHash)

		value := core.GetAnnotation(obj, v1.ResourceManagementKey)
		if value != v1.ResourceManagementDisabled {
			core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
		}
	}
}
