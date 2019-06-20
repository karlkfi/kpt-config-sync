package fake

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
)

// NamespaceConfigMutator mutates a NamespaceConfig.
type NamespaceConfigMutator func(nc *v1.NamespaceConfig)

// NamespaceConfigMeta wraps MetaMutators to be specific to NamespaceConfigs.
func NamespaceConfigMeta(opts ...object.MetaMutator) NamespaceConfigMutator {
	return func(nc *v1.NamespaceConfig) {
		mutate(nc, opts...)
	}
}

// NamespaceConfigObject initializes a NamespaceConfig.
func NamespaceConfigObject(opts ...NamespaceConfigMutator) *v1.NamespaceConfig {
	result := &v1.NamespaceConfig{TypeMeta: toTypeMeta(kinds.NamespaceConfig())}
	defaultMutate(result)
	for _, opt := range opts {
		opt(result)
	}

	return result
}
