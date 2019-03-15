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
	v1repo "github.com/google/nomos/pkg/api/policyhierarchy/v1/repo"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions/printers"
)

// InitCmd is the Cobra object representing the nomos init command
var InitCmd = &cobra.Command{
	Use:   "init DIRECTORY",
	Short: "Initialize a CSP Configuration Management directory",
	Long: `Initialize a CSP Configuration Management directory

Given an empty directory, sets up a working CSP Configuration Management directory.
Returns an error if the given directory is non-empty.`,
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

	files, err := ioutil.ReadDir(dir.String())
	if err != nil {
		return errors.Wrapf(err, "error reading %q", dir)
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return errors.Errorf("passed dir %q is not empty", dir)
		}
	}

	repoDir := repoDirectoryBuilder{dir, status.ErrorBuilder{}}
	repoDir.createFile("", readmeFile, rootReadmeContents)

	// Create system/
	repoDir.createDir(v1repo.SystemDir)
	repoDir.createSystemFile(readmeFile, systemReadmeContents)
	err = util.WriteObject(&printers.YAMLPrinter{}, dir.String(), defaultRepo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create cluster/
	// TODO: Add readme.
	repoDir.createDir(v1repo.ClusterDir)

	// Create clusterregistry/
	// TODO: Add readme.
	repoDir.createDir(v1repo.ClusterRegistryDir)

	// Create namespaces/
	// TODO: Add readme.
	repoDir.createDir(v1repo.NamespacesDir)

	// TODO(ekitson): Update this function to return MultiError instead of returning explicit nil.
	bErr := repoDir.errors.Build()
	if bErr == nil {
		return nil
	}
	return bErr
}

type repoDirectoryBuilder struct {
	root   repo.FilePath
	errors status.ErrorBuilder
}

func (d repoDirectoryBuilder) createDir(dir string) {
	newDir := filepath.Join(d.root.String(), dir)
	err := os.Mkdir(newDir, os.ModePerm)
	if err != nil {
		d.errors.Add(status.PathWrapf(err, newDir))
	}
}

func (d repoDirectoryBuilder) createFile(dir string, path string, contents string) {
	file, err := os.Create(filepath.Join(d.root.String(), dir, path))
	if err != nil {
		d.errors.Add(status.PathWrapf(err, path))
		return
	}
	_, err = file.WriteString(contents)
	if err != nil {
		d.errors.Add(status.PathWrapf(err, path))
	}
}

func (d repoDirectoryBuilder) createSystemFile(path string, contents string) {
	d.createFile(v1repo.SystemDir, path, contents)
}
