package node

const (
	// Namespace represents a leaf node in the hierarchy which is materialized as a kubernetes Namespace.
	Namespace = Type("Namespace")
	// AbstractNamespace represents a non-leaf node in the hierarchy.
	AbstractNamespace = Type("Abstract Namespace")
)

// Type represents the type of the node.
type Type string
