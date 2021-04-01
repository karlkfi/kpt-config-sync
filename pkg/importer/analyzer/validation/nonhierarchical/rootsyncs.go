package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/status"
)

// ValidateRootSync validates the content and structure of a RootSync for any
// obvious problems.
func ValidateRootSync(rs *v1alpha1.RootSync) status.Error {
	if rs.GetName() != v1alpha1.RootSyncName {
		return InvalidSyncName(rs.Name, rs)
	}
	return validateGitSpec(rs.Spec.Git, rs)
}
