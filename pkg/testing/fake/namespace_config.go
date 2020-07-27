package fake

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
)

// NamespaceConfigObject initializes a NamespaceConfig.
func NamespaceConfigObject(opts ...core.MetaMutator) *v1.NamespaceConfig {
	result := &v1.NamespaceConfig{TypeMeta: ToTypeMeta(kinds.NamespaceConfig())}
	defaultMutate(result)
	for _, opt := range opts {
		opt(result)
	}

	return result
}
