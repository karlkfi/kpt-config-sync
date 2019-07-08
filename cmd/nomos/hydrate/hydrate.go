package hydrate

import (
	"encoding/json"
	"os"
	"time"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	out string
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)
	flags.AddValidate(Cmd)

	Cmd.Flags().StringVar(&out, "output", "compiled",
		`Location to write compiled configuration to`)
}

// Cmd is the Cobra object representing the hydrate command.
//
// TODO(b/136576007): Beter error messages.
var Cmd = &cobra.Command{
	Use:    "hydrate",
	Short:  "Hydrate ",
	Args:   cobra.ExactArgs(0),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		rootDir := flags.Path.String()
		rootPath := util.GetRootOrDie(rootDir)

		opts := filesystem.ParserOpt{Validate: flags.Validate, Vet: true, Extension: &filesystem.NomosVisitorProvider{}, RootPath: rootPath}
		parser, err := parse.NewParser(opts)
		if err != nil {
			util.PrintErrAndDie(err)
		}

		var allObjects []runtime.Object

		encounteredError := false
		hydrate.ForEachCluster(parser, "", time.Time{}, func(clusterName string, configs *namespaceconfig.AllConfigs, err status.MultiError) {
			clusterEnabled := flags.AllClusters()
			for _, cluster := range flags.Clusters {
				if clusterName == cluster {
					clusterEnabled = true
				}
			}
			if !clusterEnabled {
				return
			}

			if err != nil {
				if clusterName == "" {
					clusterName = parse.UnregisteredCluster
				}
				util.PrintErrOrDie(errors.Wrapf(err, "errors for Cluster %q", clusterName))
				encounteredError = true
				return
			}

			allObjects = append(allObjects, hydrate.Flatten(configs)...)
		})
		if encounteredError {
			os.Exit(1)
		}

		err = os.MkdirAll(out, os.ModePerm)
		if err != nil {
			util.PrintErrAndDie(err)
		}
		// TODO(b/136572782): Non-flat option.
		outFile, err := os.Create(out)

		defer func() {
			err2 := outFile.Close()
			if err2 != nil {
				util.PrintErrAndDie(err2)
			}
		}()

		if err != nil {
			util.PrintErrAndDie(err)
		}
		list := &corev1.List{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
		}
		converter := runtime.DefaultUnstructuredConverter
		for _, obj := range allObjects {
			// Must convert or else fields like status automatically get written.
			unstructuredObject, err2 := converter.ToUnstructured(obj)
			if err2 != nil {
				util.PrintErrAndDie(err2)
			}
			u := &unstructured.Unstructured{Object: unstructuredObject}
			rmBadFields(u)
			raw := runtime.RawExtension{Object: u}
			list.Items = append(list.Items, raw)
		}
		unstructuredList, err := converter.ToUnstructured(list)
		if err != nil {
			util.PrintErrAndDie(err)
		}
		// TODO(b/136575755): Output full Kubernetes List for --flat.
		content, err := json.MarshalIndent(unstructuredList, "", "\t")
		if err != nil {
			util.PrintErrAndDie(err)
		}
		_, err = outFile.Write(content)
		if err != nil {
			util.PrintErrAndDie(err)
		}
	},
}

func rmBadFields(u *unstructured.Unstructured) {
	// The conversion to unstructured automatically fills these in.
	delete(u.Object, "status")
	delete(u.Object["metadata"].(map[string]interface{}), "creationTimestamp")
}
