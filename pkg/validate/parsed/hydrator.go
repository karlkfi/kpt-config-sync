package parsed

import "github.com/google/nomos/pkg/status"

// TreeHydrator defines an interface for hydrating a parsed TreeRoot.
type TreeHydrator interface {
	// Hydrate performs hydration operations on the given TreeRoot. This is
	// expected to modify the TreeRoot and/or its objects in place. Hydrate can
	// perform any validation related to its hydration and return a MultiError if
	// any validation fails.
	Hydrate(*TreeRoot) status.MultiError
}
