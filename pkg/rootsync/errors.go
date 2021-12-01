package rootsync

import (
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
)

// Errors returns the errors referred by `errorSources`.
func Errors(rs *v1beta1.RootSync, errorSources []v1beta1.ErrorSource) []v1beta1.ConfigSyncError {
	var errs []v1beta1.ConfigSyncError
	if rs == nil {
		return errs
	}

	for _, errorSource := range errorSources {
		switch errorSource {
		case v1beta1.RenderingError:
			errs = append(errs, rs.Status.Rendering.Errors...)
		case v1beta1.SourceError:
			errs = append(errs, rs.Status.Source.Errors...)
		case v1beta1.SyncError:
			errs = append(errs, rs.Status.Sync.Errors...)
		}
	}
	return errs
}
