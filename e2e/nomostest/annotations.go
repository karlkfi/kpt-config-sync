package nomostest

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/applier"
)

// CommonAnnotationKeys include the annotation keys used in both the mono-repo and multi-repo mode.
var commonAnnotationKeys []string = []string{
	v1.ClusterNameAnnotationKey,
	v1.ResourceManagementKey,
	v1.SourcePathAnnotationKey,
	v1.SyncTokenAnnotationKey,
	v1beta1.DeclaredFieldsKey,
	v1beta1.ResourceIDKey,
}

// multiRepoOnlyAnnotationKeys include the annotation keys used only in the multi-repo mode.
var multiRepoOnlyAnnotationKeys []string = []string{
	v1beta1.GitContextKey,
	v1beta1.ResourceManagerKey,
	applier.OwningInventoryKey,
}

// GetNomosAnnotationKeys returns the set of Nomos annotations that the syncer should manage.
func GetNomosAnnotationKeys(multiRepo bool) []string {
	if multiRepo {
		return append(commonAnnotationKeys, multiRepoOnlyAnnotationKeys...)
	}
	return commonAnnotationKeys
}
