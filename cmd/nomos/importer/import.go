package importer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/cloner"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/printers"
)

// Cmd exports resources in the current kubectl context into the specified directory.
var Cmd = &cobra.Command{
	Use: "import",
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := filepath.Abs(flags.Path.String())
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "failed to get absolute path"))
		}

		clientConfig, err := restconfig.NewClientConfig()
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "failed to get kubectl config"))
		}

		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "failed to get rest.Config"))
		}

		// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
		clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: restConfig.Host}

		factory := cmdutil.NewFactory(&genericclioptions.ConfigFlags{})

		discoveryClient, err := factory.ToDiscoveryClient()
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "failed to get discovery client"))
		}

		apiResources, err := cloner.ListResources(discoveryClient)
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "failed to list available API objects"))
		}

		dynamicClient, err := factory.DynamicClient()
		if err != nil {
			util.PrintErrAndDie(errors.Wrap(err, "failed to get dynamic client"))
		}

		lister := cloner.NewResourceLister(cloner.DynamicResourcer{Interface: dynamicClient})

		var objects []ast.FileObject
		eb := multierror.Builder{}
		for _, apiResource := range apiResources {
			resources, err2 := lister.List(apiResource)
			if err2 == nil {
				eb.Add(err2)
				continue
			}
			objects = append(objects, resources...)
		}

		pather := cloner.NewPather(apiResources...)
		pather.AddPaths(objects)

		printer := printers.YAMLPrinter{}
		for _, object := range objects {
			err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(object.Dir().RelativeSlashPath())), 0750)
			if err != nil {
				eb.Add(err)
				continue
			}

			file, err := os.Create(filepath.Join(dir, filepath.FromSlash(object.RelativeSlashPath())))
			if err != nil {
				eb.Add(err)
				continue
			}

			err = printer.PrintObj(object.Object, file)
			eb.Add(err)
		}
		if eb.HasErrors() {
			fmt.Println(eb.Build().Error())
			os.Exit(1)
		}
	},
}
