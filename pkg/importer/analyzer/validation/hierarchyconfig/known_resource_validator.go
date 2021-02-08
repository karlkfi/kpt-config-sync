package hierarchyconfig

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
)

// NewHierarchyConfigScopeValidator returns a Validator that complains if a passed
// HierarchyConfig includes types that are not Namespace-scoped.
func NewHierarchyConfigScopeValidator(scoper discovery.Scoper, errOnUnknown bool) nonhierarchical.Validator {
	return nonhierarchical.PerObjectValidator(func(o ast.FileObject) status.Error {
		if o.GroupVersionKind() != kinds.HierarchyConfig() {
			return nil
		}
		s, err := o.Structured()
		if err != nil {
			return err
		}
		hc := s.(*v1.HierarchyConfig)
		return validateHierarchyConfigScopes(scoper, newFileHierarchyConfig(hc, o), errOnUnknown)
	})
}

func validateHierarchyConfigScopes(scoper discovery.Scoper, hc fileHierarchyConfig, errOnUnknown bool) status.Error {
	for _, gkc := range hc.flatten() {
		scope, err := scoper.GetGroupKindScope(gkc.GK)
		if err != nil {
			if errOnUnknown {
				return err
			}
			glog.V(6).Infof("ignored error due to --no-api-server-check: %s", err)
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
