package hierarchyconfig

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// NewHierarchyConfigScopeValidator returns a Validator that complains if a passed
// HierarchyConfig includes types that are not Namespace-scoped.
func NewHierarchyConfigScopeValidator(scoper discovery.Scoper) nonhierarchical.Validator {
	return nonhierarchical.PerObjectValidator(func(o ast.FileObject) status.Error {
		if hc, isHierarchyConfig := o.Object.(*v1.HierarchyConfig); isHierarchyConfig {
			return validateHierarchyConfigScopes(scoper, newFileHierarchyConfig(hc, o))
		}
		return nil
	})
}

func validateHierarchyConfigScopes(scoper discovery.Scoper, hc fileHierarchyConfig) status.Error {
	for _, gkc := range hc.flatten() {
		scope, err := scoper.GetGroupKindScope(gkc.GK)
		if err != nil {
			return err
		}
		switch scope {
		case discovery.NamespaceScope:
			return nil
		case discovery.ClusterScope:
			return ClusterScopedResourceInHierarchyConfigError(gkc)
		default:
			return nil
		}
	}
	return nil
}

// ClusterScopedResourceInHierarchyConfigErrorCode is the error code for ClusterScopedResourceInHierarchyConfigError
const ClusterScopedResourceInHierarchyConfigErrorCode = "1046"

var clusterScopedResourceInHierarchyConfigError = status.NewErrorBuilder(ClusterScopedResourceInHierarchyConfigErrorCode)

// ClusterScopedResourceInHierarchyConfigError reports that a Resource defined in a HierarchyConfig
// has Cluster scope which means it's not feasible to interpret the resource in a hierarchical
// manner
func ClusterScopedResourceInHierarchyConfigError(config id.HierarchyConfig) status.Error {
	gk := config.GroupKind()
	return clusterScopedResourceInHierarchyConfigError.
		Sprintf("This HierarchyConfig references the APIResource %q which has Cluster scope. "+
			"Cluster scoped objects are not permitted in HierarchyConfig.",
			gk.String()).
		BuildWithResources(config)
}
