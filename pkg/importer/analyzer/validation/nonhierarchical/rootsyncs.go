package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/status"
)

// ValidateRootSync validates the content and structure of a RootSync for any
// obvious problems.
func ValidateRootSync(rs *v1beta1.RootSync) status.Error {
	if rs.GetName() != configsync.RootSyncName {
		return InvalidSyncName(configsync.RootSyncName, rs)
	}
	return validateGitSpec(rs.Spec.Git, rs)
}
