package action

// Interface represents a CUD action on a kubernetes resource
type Interface interface {
	// Operation returns the type of operation
	Operation() OperationType
	// Execute will execute the operation then return an error on failure
	Execute() error
	// Resource returns the type of resource being operated on
	Resource() string
	// Kind returns the kind of the resource being operated on
	Kind() string
	// Namespace returns the namespace of the resource being operated on
	Namespace() string
	// Group returns the group of the resource being operated on
	Group() string
	// Version returns the version of the resource being operated on
	Version() string
	// Name returns the name of the resource being operated on
	Name() string
	// String representation of this action. It should uniquely identify the resource being modified,
	// (group, version, kind, namespace, name).
	String() string
}
