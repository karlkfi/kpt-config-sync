package fake

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
)

// ClusterConfigMutator mutates a ClusterConfig.
type ClusterConfigMutator func(cc *v1.ClusterConfig)

// ClusterConfigMeta wraps a MetaMutator for modifying ClusterConfigs.
func ClusterConfigMeta(opts ...object.MetaMutator) ClusterConfigMutator {
	return func(cc *v1.ClusterConfig) {
		mutate(cc, opts...)
	}
}

// CRDClusterConfigObject initializes a valid CRDClusterConfig.
func CRDClusterConfigObject(opts ...ClusterConfigMutator) *v1.ClusterConfig {
	return ClusterConfigObject(ClusterConfigMeta(object.Name(v1.CRDClusterConfigName)))
}

// ClusterConfigObject initializes a ClusterConfig.
func ClusterConfigObject(opts ...ClusterConfigMutator) *v1.ClusterConfig {
	result := &v1.ClusterConfig{TypeMeta: toTypeMeta(kinds.ClusterConfig())}
	defaultMutate(result)
	mutate(result, object.Name(v1.ClusterConfigName))
	for _, opt := range opts {
		opt(result)
	}

	return result
}
