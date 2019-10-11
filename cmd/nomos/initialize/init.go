package initialize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/repo"
	"github.com/google/nomos/cmd/nomos/util"
	v1repo "github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions/printers"
)

var force bool

func init() {
	flags.AddPath(Cmd)
	Cmd.Flags().BoolVar(&force, "force", false,
		"write to directory even if nonempty, overwriting conflicting files")
}

// Cmd is the Cobra object representing the nomos init command
var Cmd = &cobra.Command{
	Use:   "init DIRECTORY",
	Short: "Initialize a Anthos Configuration Management directory",
	Long: `Initialize a Anthos Configuration Management directory

Set up a working Anthos Configuration Management directory with a default Repo object, documentation,
and directories.

By default, does not initialize directories containing files. Use --force to
initialize nonempty directories.`,
	Example: `  nomos init
  nomos init --path=my/directory
  nomos init --path=/path/to/my/directory`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		err := Initialize(flags.Path)
		if err != nil {
			util.PrintErrAndDie(err)
		}
	},
}

// Initialize initializes a Nomos directory
func Initialize(dir repo.FilePath) error {
	if !dir.Exists {
		err := os.MkdirAll(dir.String(), os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "unable to create dir %q", dir)
		}
	}

	if !force {
		err := checkEmpty(dir)
		if err != nil {
			return err
		}
	}

	repoDir := repoDirectoryBuilder{root: dir}
	repoDir.createFile("", readmeFile, rootReadmeContents)

	// Create system/
	repoDir.createDir(v1repo.SystemDir)
	repoDir.createSystemFile(readmeFile, systemReadmeContents)
	repoObj, err := defaultRepo()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = util.WriteObject(&printers.YAMLPrinter{}, dir.String(), repoObj)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create cluster/
	repoDir.createDir(v1repo.ClusterDir)

	// Create clusterregistry/
	repoDir.createDir(v1repo.ClusterRegistryDir)

	// Create namespaces/
	repoDir.createDir(v1repo.NamespacesDir)

	return repoDir.errors
}

func checkEmpty(dir repo.FilePath) error {
	files, err := ioutil.ReadDir(dir.String())
	if err != nil {
		return errors.Wrapf(err, "error reading %q", dir)
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return errors.Errorf("passed dir %q is not empty; use --force to proceed.", dir)
		}
	}
	return nil
}

type repoDirectoryBuilder struct {
	root   repo.FilePath
	errors status.MultiError
}

func (d repoDirectoryBuilder) createDir(dir string) {
	newDir := filepath.Join(d.root.String(), dir)
	err := os.Mkdir(newDir, os.ModePerm)
	if err != nil {
		d.errors = status.Append(d.errors, status.PathWrapf(err, newDir))
	}
}

func (d repoDirectoryBuilder) createFile(dir string, path string, contents string) {
	file, err := os.Create(filepath.Join(d.root.String(), dir, path))
	if err != nil {
		d.errors = status.Append(d.errors, status.PathWrapf(err, path))
		return
	}
	_, err = file.WriteString(contents)
	if err != nil {
		d.errors = status.Append(d.errors, status.PathWrapf(err, path))
	}
}

func (d repoDirectoryBuilder) createSystemFile(path string, contents string) {
	d.createFile(v1repo.SystemDir, path, contents)
}
