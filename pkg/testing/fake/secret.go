package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	corev1 "k8s.io/api/core/v1"
)

// SecretObject returns an initialized Secret.
func SecretObject(name string, opts ...core.MetaMutator) *corev1.Secret {
	result := &corev1.Secret{TypeMeta: ToTypeMeta(kinds.Secret())}
	mutate(result, core.Name(name))
	mutate(result, opts...)

	return result
}
