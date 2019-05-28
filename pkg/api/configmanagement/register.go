package configmanagement

const (
	// CLIName is the short name of the CLI.
	CLIName = "nomos"

	// MetricsNamespace is the namespace that metrics are held in.
	MetricsNamespace = "gkeconfig"

	// OperatorKind is the Kind of the Operator config object.
	OperatorKind = "ConfigManagement"

	// GroupName is the name of the group of configmanagement resources.
	GroupName = "configmanagement.gke.io"

	// ProductName is what we call Nomos externally.
	ProductName = "Anthos Configuration Management"

	// ControllerNamespace is the Namespace used for Nomos controllers
	ControllerNamespace = "config-management-system"
)
