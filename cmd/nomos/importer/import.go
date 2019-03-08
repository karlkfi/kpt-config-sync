package importer

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/cloner"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/printers"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// Cmd exports resources in the current kubectl context into the specified directory.
var Cmd = &cobra.Command{
	Use: "import",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Allow other outputs than os.Stderr.
		errOutput := cloner.NewStandardErrorOutput()
		dir, err := filepath.Abs(flags.Path.String())
		errOutput.AddAndDie(errors.Wrap(err, "failed to get absolute path"))

		clientConfig, err := restconfig.NewClientConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get kubectl config"))

		restConfig, err := clientConfig.ClientConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get rest.Config"))

		// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
		clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: restConfig.Host}

		factory := cmdutil.NewFactory(&genericclioptions.ConfigFlags{})

		discoveryClient, err := factory.ToDiscoveryClient()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get discovery client"))

		apiResources := cloner.ListResources(discoveryClient, errOutput)
		errOutput.DieIfPrintedErrors("failed to list available API objects")

		dynamicClient, err := factory.DynamicClient()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get dynamic client"))

		lister := cloner.NewResourceLister(cloner.DynamicResourcer{Interface: dynamicClient})

		var objects []ast.FileObject
		for _, apiResource := range apiResources {
			resources := lister.List(apiResource, errOutput)
			objects = append(objects, resources...)
		}

		pather := cloner.NewPather(apiResources...)
		pather.AddPaths(objects)

		printer := &printers.YAMLPrinter{}
		for _, object := range objects {
			err2 := writeObject(printer, dir, object)
			errOutput.Add(err2)
		}
		errOutput.DieIfPrintedErrors("encountered errors writing resources to files")
	},
}

func writeObject(printer printers.ResourcePrinter, dir string, object ast.FileObject) error {
	if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(object.Dir().RelativeSlashPath())), 0750); err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(dir, filepath.FromSlash(object.RelativeSlashPath())))
	if err != nil {
		return err
	}

	return printer.PrintObj(object.Object, file)
}
