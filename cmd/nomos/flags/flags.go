package flags

import (
	"github.com/spf13/cobra"
)

const (
	// pathFlag is the flag to set the Path of the Nomos directory.
	pathFlag = "path"

	// PathDefault is the default value of the path flag if unset.
	PathDefault = "."

	// contextsFlag is the flag name for the Contexts below.
	contextsFlag = "contexts"

	// clusterFlag is the flag name for the Clusters below.
	clustersFlag = "clusters"

	// SkipAPIServerFlag is the flag name for SkipAPIServer below.
	SkipAPIServerFlag = "no-api-server-check"
)

var (
	// Contexts contains the list of .kubeconfig contexts that are targets of cross-cluster
	// commands.
	Contexts []string

	// Clusters contains the list of Cluster names (specified in clusters/) to perform an action on.
	Clusters []string

	// Path says where the Nomos directory is
	Path string

	// SkipAPIServer directs whether to try to contact the API Server for checks.
	SkipAPIServer bool
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
	cmd.Flags().StringVar(&Path, pathFlag, PathDefault,
		`Root directory to use as a Anthos Configuration Management repository.`)
}

// AddSkipAPIServerCheck adds the --no-api-server-check flag.
func AddSkipAPIServerCheck(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&SkipAPIServer, SkipAPIServerFlag, false,
		"If true, disables talking to the API Server for discovery.")
}

// AllClusters returns true if all clusters should be processed.
func AllClusters() bool {
	return Clusters == nil
}
