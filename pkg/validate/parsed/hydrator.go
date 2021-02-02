package parsed

import "github.com/google/nomos/pkg/status"

// Hydrator defines an interface for hydrating a parsed Root.
type Hydrator interface {
	// Hydrate performs hydration operations on the given Root. This is expected
	// to modify the Root and/or its objects in place. Hydrate can perform any
	// validation related to its hydration and return a MultiError if any
	// validation fails.
	Hydrate(Root) status.MultiError
}

// FlatHydrator defines an interface for hydrating a parsed FlatRoot.
type FlatHydrator interface {
	// Hydrate is Hydrator.Hydrate for a FlatRoot.
	Hydrate(*FlatRoot) status.MultiError
}

type flatHydrator struct {
	hydrator Hydrator
}

var _ FlatHydrator = &flatHydrator{}

// Hydrate implements FlatHydrator.
func (f *flatHydrator) Hydrate(root *FlatRoot) status.MultiError {
	return f.hydrator.Hydrate(root)
}

// FlatWrap returns a FlatHydrator that wraps the given base Hydrator.
func FlatWrap(base Hydrator) FlatHydrator {
	return &flatHydrator{base}
}

// TreeHydrator defines an interface for hydrating a parsed TreeRoot.
type TreeHydrator interface {
	// Hydrate is Hydrator.Hydrate for a TreeRoot.
	Hydrate(*TreeRoot) status.MultiError
}

type treeHydrator struct {
	hydrator Hydrator
}

var _ TreeHydrator = &treeHydrator{}

// Hydrate implements TreeHydrator.
func (f *treeHydrator) Hydrate(root *TreeRoot) status.MultiError {
	return f.hydrator.Hydrate(root)
}

// TreeWrap returns a TreeHydrator that wraps the given base Hydrator.
func TreeWrap(base Hydrator) TreeHydrator {
	return &treeHydrator{base}
}
