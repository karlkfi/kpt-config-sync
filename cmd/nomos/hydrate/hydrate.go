package hydrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	flat      bool
	outPath   string
	extension string

	converter = runtime.DefaultUnstructuredConverter
)

func init() {
	flags.AddClusters(Cmd)
	flags.AddPath(Cmd)

	Cmd.Flags().BoolVar(&flat, "flat", false,
		`If enabled, print all output to a single file`)
	Cmd.Flags().StringVar(&outPath, "output", "compiled",
		`Location to write hydrated configuration to.

If --flat is not enabled, writes each resource manifest as a
separate file. You may run "kubectl apply -fR" on the result to apply
the configuration to a cluster. If the repository declares any Cluster
resources, contains a subdirectory for each Cluster.

If --flat is enabled, writes to the, writes a single file holding all
resource manifests. You may run "kubectl apply -f" on the result to
apply the configuration to a cluster.`)
	Cmd.Flags().StringVar(&extension, "format", "yaml",
		`Output format of hydrated configuration. Accepts 'yaml' and 'json'.`)
}

// Cmd is the Cobra object representing the hydrate command.
//
// TODO(b/136576007): Better error messages.
var Cmd = &cobra.Command{
	Use:   "hydrate",
	Short: "Compiles the local repository to the exact form that would be sent to the APIServer.",
	Long: `Compiles the local repository to the exact form that would be sent to the APIServer.

The output directory consists of one directory per declared Cluster, and defaultcluster/ for
clusters without declarations. Each directory holds the full set of manifests which you could
kubectl apply -fR to the cluster, and is equivalent to the state ConfigManagement maintains on your
clusters.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		switch extension {
		case "yaml", "json": // do nothing
		default:
			util.PrintErrAndDie(errors.New("format must argument be 'yaml' or 'json'"))
		}

		rootDir := flags.Path.String()
		rootPath := util.GetRootOrDie(rootDir)

		opts := filesystem.ParserOpt{Extension: &filesystem.NomosVisitorProvider{}, RootPath: rootPath}
		parser, err := parse.NewParser(opts)
		if err != nil {
			util.PrintErrAndDie(err)
		}

		var allObjects []runtime.Object

		encounteredError := false
		numClusters := 0
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
			numClusters++

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

		multiCluster := numClusters > 1
		fileObjects := hydrate.ToFileObjects(extension, multiCluster, allObjects...)
		if flat {
			printFlatOutput(fileObjects)
		} else {
			printDirectoryOutput(fileObjects)
		}

		if encounteredError {
			os.Exit(1)
		}

		if err != nil {
			util.PrintErrAndDie(err)
		}
	},
}

func printFlatOutput(fileObjects []ast.FileObject) {
	var objects []*unstructured.Unstructured
	for _, o := range fileObjects {
		objects = append(objects, toUnstructured(o.Object))
	}

	printFile(outPath, objects)
}

func printDirectoryOutput(fileObjects []ast.FileObject) {
	// TODO: Make compatible with multi-cluster.
	files := make(map[string][]*unstructured.Unstructured)
	for _, obj := range fileObjects {
		files[obj.SlashPath()] = append(files[obj.SlashPath()], toUnstructured(obj.Object))
	}

	for file, objects := range files {
		printFile(filepath.Join(outPath, file), objects)
	}
}

func toUnstructured(o runtime.Object) *unstructured.Unstructured {
	// Must convert or else fields like status automatically get written.
	unstructuredObject, err2 := converter.ToUnstructured(o)
	if err2 != nil {
		util.PrintErrAndDie(err2)
	}
	u := &unstructured.Unstructured{Object: unstructuredObject}
	rmBadFields(u)
	return u
}

func printFile(file string, objects []*unstructured.Unstructured) {
	err := os.MkdirAll(filepath.Dir(file), os.ModePerm)
	if err != nil {
		util.PrintErrAndDie(err)
	}

	outFile, err := os.Create(file)
	if err != nil {
		util.PrintErrAndDie(err)
	}

	defer func() {
		err2 := outFile.Close()
		if err2 != nil {
			util.PrintErrAndDie(err2)
		}
	}()

	var content string
	switch extension {
	case "yaml":
		content = toYAML(objects)
	case "json":
		content = toJSON(objects)
	}
	_, err3 := outFile.WriteString(content)
	if err3 != nil {
		util.PrintErrAndDie(err3)
	}
}

func toYAML(objects []*unstructured.Unstructured) string {
	content := strings.Builder{}
	for _, object := range objects {
		content.WriteString("---\n")
		bytes, err2 := yaml.Marshal(object.Object)
		if err2 != nil {
			util.PrintErrAndDie(err2)
		}
		content.Write(bytes)
	}
	return content.String()
}

func toJSON(objects []*unstructured.Unstructured) string {
	list := &corev1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
	}
	for _, obj := range objects {
		u := toUnstructured(obj)
		raw := runtime.RawExtension{Object: u}
		list.Items = append(list.Items, raw)
	}
	unstructuredList, err := converter.ToUnstructured(list)
	if err != nil {
		util.PrintErrAndDie(err)
	}
	content, err := json.MarshalIndent(unstructuredList, "", "\t")
	if err != nil {
		util.PrintErrAndDie(err)
	}
	return string(content)
}

func rmBadFields(u *unstructured.Unstructured) {
	// The conversion to unstructured automatically fills these in.
	delete(u.Object, "status")
	delete(u.Object["metadata"].(map[string]interface{}), "creationTimestamp")
}
