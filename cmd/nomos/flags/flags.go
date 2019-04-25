package flags

import (
	"github.com/google/nomos/cmd/nomos/repo"
	"github.com/spf13/cobra"
)

const (
	// pathFlag is the flag to set the Path of the Nomos directory.
	pathFlag = "path"

	// validateFlag is the flag to set the Validate value
	validateFlag = "validate"

	// ClustersName is the flag name for the Clusters below.
	ClustersName = "clusters"

	// AllClustersName is the flag name for AllClusters below.
	AllClustersName = "all-clusters"
)

// AddPath adds the --path flag
func AddPath(cmd *cobra.Command) {
	cmd.Flags().Var(&Path, pathFlag,
		`Root directory to use as a CSP Configuration Management repository.`)
}

// AddValidate adds the --validate flag for whether to use schemas for
// validating
func AddValidate(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&Validate, validateFlag, true,
		`If true, use a schema to validate the CSP Configuration Management repository.`)
}

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
