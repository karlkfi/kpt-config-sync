package discovery

import "k8s.io/apimachinery/pkg/runtime/schema"

// Scoper returns the ObjectScope of a provided GroupKind.
type Scoper interface {
	GetScope(gk schema.GroupKind) ObjectScope
}
