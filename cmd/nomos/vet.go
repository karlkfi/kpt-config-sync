package nomos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
)

func init() {
	rootCmd.AddCommand(vetCmd)

	vetCmd.Flags().BoolVar(&validate, "validate", true, "If true, use a schema to validate the input")
	vetCmd.Flags().BoolVar(&printGenerated, "print", false, "If true, print generated Nomos CRDs")
}

var validate bool
var printGenerated bool

var vetCmd = &cobra.Command{
	Use:   "vet DIRECTORY",
	Short: "Validate a GKE Policy Management directory",
	Long: `Validate a GKE Policy Management directory

Checks for semantic and syntactic errors in a GKE Policy Management directory
that will interfere with applying resources. Prints found errors to STDERR and
returns a non-zero error code if any issues are found.
`,
	Example: `  nomos vet my/repo
  nomos vet /path/to/my/repo`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Check for a set environment variable instead of using a flag so as not to expose
		// this WIP externally.
		_, bespin := os.LookupEnv("NOMOS_ENABLE_BESPIN")

		dir, err := filepath.Abs(args[0])
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to get absolute path"))
		}

		clientConfig, err := restconfig.NewClientConfig()
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to get kubectl config"))
		}

		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to get rest.Config"))
		}

		client, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to create client"))
		}

		// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
		clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: restConfig.Host}
		p, err := filesystem.NewParser(
			&genericclioptions.ConfigFlags{}, client.Discovery(), filesystem.ParserOpt{Validate: validate, Vet: true, Bespin: bespin})
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to create parser"))
		}

		resources, err := p.Parse(dir)
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Found issues"))
		}
		if printGenerated {
			err := prettyPrint(resources)
			if err != nil {
				printErrAndDie(errors.Wrap(err, "Failed to print generated CRDs"))
			}
		}
	},
}

func printErrAndDie(err error) {
	// nolint: errcheck
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func prettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return err
}
