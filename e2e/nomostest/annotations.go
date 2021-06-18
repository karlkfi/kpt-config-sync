package nomostest

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/constants"
)

// CommonAnnotationKeys include the annotation keys used in both the mono-repo and multi-repo mode.
var commonAnnotationKeys = []string{
	v1.ClusterNameAnnotationKey,
	v1.ResourceManagementKey,
	v1.SourcePathAnnotationKey,
	v1.SyncTokenAnnotationKey,
	constants.DeclaredFieldsKey,
	constants.ResourceIDKey,
}

// multiRepoOnlyAnnotationKeys include the annotation keys used only in the multi-repo mode.
var multiRepoOnlyAnnotationKeys = []string{
	constants.GitContextKey,
	constants.ResourceManagerKey,
	applier.OwningInventoryKey,
}

// GetNomosAnnotationKeys returns the set of Nomos annotations that the syncer should manage.
func GetNomosAnnotationKeys(multiRepo bool) []string {
	if multiRepo {
		return append(commonAnnotationKeys, multiRepoOnlyAnnotationKeys...)
	}
	return commonAnnotationKeys
}
