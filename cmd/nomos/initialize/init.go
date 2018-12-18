package initialize

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/repo"
	"github.com/google/nomos/cmd/nomos/util"
	v1repo "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// InitCmd is the Cobra object representing the nomos init command
var InitCmd = &cobra.Command{
	Use:   "init DIRECTORY",
	Short: "Initialize a GKE Policy Management directory",
	Long: `Initialize a GKE Policy Management directory

Given an empty directory, sets up a working GKE Policy Management directory.
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

	repoDir := repoDirectoryBuilder{dir, multierror.Builder{}}
	repoDir.createFile("", readmeFile, rootReadmeContents)

	// Create system/
	repoDir.createDir(v1repo.SystemDir)
	repoDir.createSystemFile(readmeFile, systemReadmeContents)
	repoDir.createSystemFile(repoFile, repoContents)
	repoDir.createSystemFile(rbacSyncFile, rbacSyncContents)
	repoDir.createSystemFile(resourceQuotaSyncFile, resourceQuotaSyncContents)
	repoDir.createSystemFile(podSecuritySyncFile, podSecuritySyncContents)

	// Create cluster/
	repoDir.createDir(v1repo.ClusterDir)
	repoDir.createFile(v1repo.ClusterDir, readmeFile, clusterReadmeContents)

	// Create namespaces/
	repoDir.createDir(v1repo.NamespacesDir)
	repoDir.createFile(v1repo.NamespacesDir, readmeFile, namespacesReadmeContents)

	return repoDir.errors.Build()
}

type repoDirectoryBuilder struct {
	root   repo.FilePath
	errors multierror.Builder
}

func (d repoDirectoryBuilder) createDir(dir string) {
	newDir := filepath.Join(d.root.String(), dir)
	err := os.Mkdir(newDir, os.ModePerm)
	if err != nil {
		d.errors.Add(errors.Wrapf(err, "unable to create directory %q", newDir))
	}
}

func (d repoDirectoryBuilder) createFile(dir string, path string, contents string) {
	file, err := os.Create(filepath.Join(d.root.String(), dir, path))
	if err != nil {
		d.errors.Add(errors.Wrapf(err, "error creating file %q", path))
		return
	}
	_, err = file.WriteString(contents)
	if err != nil {
		d.errors.Add(errors.Wrapf(err, "error writing to file %q", path))
	}
}

func (d repoDirectoryBuilder) createSystemFile(path string, contents string) {
	d.createFile(v1repo.SystemDir, path, contents)
}
