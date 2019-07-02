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

	// contextsFlag is the flag name for the Contexts below.
	contextsFlag = "contexts"

	// clusterFlag is the flag name for the Clusters below.
	clustersFlag = "clusters"
)

var (
	// Contexts contains the list of .kubeconfig contexts that are targets of cross-cluster
	// commands.
	Contexts []string

	// Clusters contains the list of Cluster names (specified in clusters/) to perform an action on.
	Clusters []string

	// Path says where the Nomos directory is
	Path = repo.WorkingDirectoryPath

	// Validate determines whether to use a schema to validate the input
	Validate bool
)

// AddContexts adds the --contexts flag.
func AddContexts(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&Contexts, contextsFlag, nil,
		`Accepts a comma-separated list of contexts to use in multi-cluster commands. Defaults to all contexts. Use "" for no contexts.`)
}

// AddClusters adds the --clusters flag.
func AddClusters(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&Clusters, clustersFlag, nil,
		`Accepts a comma-separated list of Cluster names to use in multi-cluster commands. Defaults to all clusters. Use "" for no clusters.`)
}

// AddPath adds the --path flag.
func AddPath(cmd *cobra.Command) {
	cmd.Flags().Var(&Path, pathFlag,
		`Root directory to use as a Anthos Configuration Management repository.`)
}

// AddValidate adds the --validate flag for whether to use schemas for
// validating
func AddValidate(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&Validate, validateFlag, true,
		`If true, use a schema to validate the Anthos Configuration Management repository.`)
}
