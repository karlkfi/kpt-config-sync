package filesystem

import (
	"time"

	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

// configParser defines the minimum interface required for Reconciler to use a Parser to read
// configs from a filesystem.
type configParser interface {
	Parse(
		importToken string,
		currentConfigs *namespaceconfig.AllConfigs,
		loadTime time.Time,
		clusterName string,
	) (*namespaceconfig.AllConfigs, status.MultiError)
}
