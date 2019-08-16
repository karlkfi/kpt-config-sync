package filesystem

import (
	"time"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
)

// ConfigParser defines the minimum interface required for Reconciler to use a Parser to read
// configs from a filesystem.
type ConfigParser interface {
	Parse(
		importToken string,
		currentConfigs *namespaceconfig.AllConfigs,
		loadTime time.Time,
		clusterName string,
	) (*namespaceconfig.AllConfigs, status.MultiError)

	// ReadClusterRegistryResources returns the list of Clusters contained in the repo.
	ReadClusterRegistryResources() []ast.FileObject
}
