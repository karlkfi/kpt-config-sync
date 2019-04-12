package flags

import "github.com/google/nomos/cmd/nomos/repo"

const (
	// ValidateFlag is the flag to set the Validate value
	ValidateFlag = "validate"
	// PathFlag is the flag to set the Path value
	PathFlag = "path"

	// ClustersName is the flag name for the Clusters below.
	ClustersName = "clusters"

	// AllClustersName is the flag name for AllClusters below.
	AllClustersName = "all-clusters"
)

var (
	// Validate determines whether to use a schema to validate the input
	Validate bool

	// Path says where the Nomos directory is
	Path = repo.WorkingDirectoryPath

	// Clusters contains the list of clusters that are targets of cross-cluster
	// commands.
	Clusters []string

	// AllClusters is set if the user wants a multi-cluster command to take
	// effect on all clusters from the user's Kubernetes configuration.
	AllClusters bool
)
