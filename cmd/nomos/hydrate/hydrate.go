package hydrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/validate"
	"github.com/google/nomos/pkg/vet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	flags.AddSkipAPIServerCheck(Cmd)

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
	RunE: func(cmd *cobra.Command, args []string) error {
		// Don't show usage on error, as argument validation passed.
		cmd.SilenceUsage = true

		switch extension {
		case "yaml", "json": // do nothing
		default:
			return errors.New("format must argument be 'yaml' or 'json'")
		}

		abs, err := filepath.Abs(flags.Path)
		if err != nil {
			return err
		}
		rootDir, err := cmpath.AbsoluteOS(abs)
		if err != nil {
			return err
		}
		policyDir := cmpath.RelativeOS(flags.Path)

		files, err := parse.FindFiles(rootDir)
		if err != nil {
			return err
		}
		// Hydrate is only used in hierarchical mode.
		files = filesystem.FilterHierarchyFiles(rootDir, files)

		filePaths := reader.FilePaths{
			RootDir:   rootDir,
			PolicyDir: policyDir,
			Files:     files,
		}

		dc, err := importer.DefaultCLIOptions.ToDiscoveryClient()
		if err != nil {
			return err
		}
		parser := filesystem.NewParser(&reader.File{})

		crds, err := parse.GetSyncedCRDs(cmd.Context(), flags.SkipAPIServer)
		if err != nil {
			return err
		}
		addFunc := vet.AddCachedAPIResources(rootDir.Join(vet.APIResourcesPath))

		var converter *declared.ValueConverter
		if !flags.SkipAPIServer {
			converter, err = declared.NewValueConverter(dc)
			if err != nil {
				return err
			}
		}

		options := validate.Options{
			PolicyDir:         policyDir,
			PreviousCRDs:      crds,
			BuildScoper:       discovery.ScoperBuilder(dc, addFunc),
			Converter:         converter,
			AllowUnknownKinds: flags.SkipAPIServer,
		}

		var allObjects []ast.FileObject
		encounteredError := false
		numClusters := 0
		hydrate.ForEachCluster(parser, options, filePaths, func(clusterName string, fileObjects []ast.FileObject, err status.MultiError) {
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

			allObjects = append(allObjects, fileObjects...)
		})

		multiCluster := numClusters > 1
		fileObjects := hydrate.GenerateUniqueFileNames(extension, multiCluster, allObjects...)
		if flat {
			err = printFlatOutput(fileObjects)
		} else {
			err = printDirectoryOutput(fileObjects)
		}
		if err != nil {
			return err
		}

		if encounteredError {
			os.Exit(1)
		}

		return nil
	},
}

func printFlatOutput(fileObjects []ast.FileObject) error {
	objects := make([]*unstructured.Unstructured, len(fileObjects))
	for i, o := range fileObjects {
		objects[i] = o.Unstructured
	}

	return printFile(outPath, objects)
}

func printDirectoryOutput(fileObjects []ast.FileObject) error {
	files := make(map[string][]*unstructured.Unstructured)
	for _, obj := range fileObjects {
		u, err := toUnstructured(obj.Unstructured)
		if err != nil {
			return err
		}
		files[obj.SlashPath()] = append(files[obj.SlashPath()], u)
	}

	for file, objects := range files {
		err := printFile(filepath.Join(outPath, file), objects)
		if err != nil {
			return err
		}
	}
	return nil
}

func toUnstructured(o client.Object) (*unstructured.Unstructured, error) {
	// Must convert or else fields like status automatically get written.
	unstructuredObject, err2 := converter.ToUnstructured(o)
	if err2 != nil {
		return nil, err2
	}
	u := &unstructured.Unstructured{Object: unstructuredObject}
	rmBadFields(u)
	return u, nil
}

func printFile(file string, objects []*unstructured.Unstructured) (err error) {
	err = os.MkdirAll(filepath.Dir(file), os.ModePerm)
	if err != nil {
		return err
	}

	outFile, err := os.Create(file)
	if err != nil {
		return err
	}

	defer func() {
		err2 := outFile.Close()
		if err2 != nil && err == nil {
			// Assign the named parameter since there's no other way to ensure we get
			// the error from the deferred Close.
			err = err2
		}
	}()

	var content string
	switch extension {
	case "yaml":
		content, err = toYAML(objects)
	case "json":
		content, err = toJSON(objects)
	}
	if err != nil {
		return err
	}
	_, err = outFile.WriteString(content)
	return err
}

func toYAML(objects []*unstructured.Unstructured) (string, error) {
	content := strings.Builder{}
	for _, o := range objects {
		content.WriteString("---\n")
		bytes, err := yaml.Marshal(o.Object)
		if err != nil {
			return "", err
		}
		content.Write(bytes)
	}
	return content.String(), nil
}

func toJSON(objects []*unstructured.Unstructured) (string, error) {
	list := &corev1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
	}
	for _, obj := range objects {
		u, err := toUnstructured(obj)
		if err != nil {
			return "", err
		}
		raw := runtime.RawExtension{Object: u}
		list.Items = append(list.Items, raw)
	}
	unstructuredList, err := converter.ToUnstructured(list)
	if err != nil {
		return "", err
	}
	content, err := json.MarshalIndent(unstructuredList, "", "\t")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func rmBadFields(u *unstructured.Unstructured) {
	// The conversion to unstructured automatically fills these in.
	delete(u.Object, "status")
	delete(u.Object["metadata"].(map[string]interface{}), "creationTimestamp")
}
