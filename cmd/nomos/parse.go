package nomos

import (
	"fmt"
	"os"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
)

// parse parses a GKE Policy Directory with a Parser using the specified Parser optional arguments.
// Exits early if it encounters parsing/validation errors.
func parse(dir string, parserOpt filesystem.ParserOpt) *v1.AllPolicies {
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
		&genericclioptions.ConfigFlags{}, client.Discovery(), parserOpt)
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to create parser"))
	}

	resources, err := p.Parse(dir)
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Found issues"))
	}

	return resources
}

func printErrAndDie(err error) {
	// nolint: errcheck
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
