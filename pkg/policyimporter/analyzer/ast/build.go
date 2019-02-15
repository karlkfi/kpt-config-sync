package ast

// BuildOpt performs some operation on Root, returning error if some problem is found.
type BuildOpt func(root *Root) error

// Build constructs a Root by starting with an empty Root and performing the supplied BuildOpts it,
// returning the result. If any errors are encountered, returns error.
func Build(opts ...BuildOpt) (*Root, error) {
	root := &Root{}

	for _, opt := range opts {
		err := opt(root)
		if err != nil {
			return nil, err
		}
	}

	return root, nil
}
