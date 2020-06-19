package nomostest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	appsv1 "k8s.io/api/apps/v1"
)

const manifests = "manifests"

// baseDir is the path to the Nomos repository root from test case files.
//
// All paths must be relative to the test file that is running. There is probably
// a more elegant way to do this.
var baseDir = filepath.FromSlash("../..")

var manifestsDir = filepath.Join(baseDir, manifests)
var templateDir = filepath.Join(manifestsDir, "templates")

// e2e/raw-nomos/manifests/default-configmaps.yaml
const defaultConfigMapsName = "default-configmaps.yaml"

var defaultConfigMaps = filepath.Join(baseDir, "e2e", "raw-nomos", manifests, defaultConfigMapsName)

var templates = []string{
	"git-importer.yaml",
	"monitor.yaml",
}

// installConfigSync installs ConfigSync on the test cluster, and returns a
// callback for checking that the installation succeeded.
func installConfigSync(nt *NT) func() error {
	nt.T.Helper()
	tmpManifestsDir := filepath.Join(nt.TmpDir, manifests)

	objs := installationManifests(nt, tmpManifestsDir)

	for _, o := range objs {
		if o.GroupVersionKind() == kinds.Namespace() && o.GetName() == configmanagement.ControllerNamespace {
			// We've already created the config-management-system Namespace.
			continue
		}
		err := nt.Create(o)
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	return func() error {
		return Retry(60*time.Second, func() error {
			err := nt.Validate("monitor", configmanagement.ControllerNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
			if err != nil {
				return err
			}
			return nt.Validate("git-importer", configmanagement.ControllerNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
		})
	}
}

func copyFile(src, dst string) error {
	bytes, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, bytes, fileMode)
}

// installationManifests generates the ConfigSync installation YAML and copies
// it to the test's temporary directory.
func installationManifests(nt *NT, tmpManifestsDir string) []core.Object {
	nt.T.Helper()

	manifestFiles, err := ioutil.ReadDir(manifestsDir)
	if err != nil {
		nt.T.Fatal(err)
	}
	err = os.MkdirAll(tmpManifestsDir, fileMode)
	if err != nil {
		nt.T.Fatal(err)
	}
	for _, f := range manifestFiles {
		// Explicitly not recursive since we want to treat the template files specially.
		if !f.IsDir() {
			from := filepath.Join(manifestsDir, f.Name())
			to := filepath.Join(tmpManifestsDir, f.Name())
			err = copyFile(from, to)
			nt.T.Logf("copying %q to %q", from, to)
			if err != nil {
				nt.T.Fatal(err)
			}
		}
	}

	err = copyFile(defaultConfigMaps, filepath.Join(tmpManifestsDir, defaultConfigMapsName))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Generate the Deployment YAML.
	for _, template := range templates {
		// It isn't strictly necessary for us to have the YAML file
		// (we could apply directly), but it is very helpful for debugging.
		bytes, err := ioutil.ReadFile(filepath.Join(templateDir, template))
		if err != nil {
			nt.T.Fatal(err)
		}
		replaced := strings.Replace(string(bytes),
			"IMAGE_NAME", "localhost:5000/nomos:latest", -1)

		err = ioutil.WriteFile(filepath.Join(tmpManifestsDir, template), []byte(replaced), fileMode)
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	// Create the list of paths for the FileReader to read.
	readPath, err := cmpath.AbsoluteOS(tmpManifestsDir)
	if err != nil {
		nt.T.Fatal(err)
	}
	files, err := ioutil.ReadDir(tmpManifestsDir)
	if err != nil {
		nt.T.Fatal(err)
	}
	paths := make([]cmpath.Absolute, len(files))
	for i, f := range files {
		paths[i] = readPath.Join(cmpath.RelativeSlash(f.Name()))
	}
	// Read the manifests cached in the tmpdir.
	reader := filesystem.FileReader{}
	fos, err := reader.Read(readPath, paths)
	if err != nil {
		nt.T.Fatal(err)
	}

	var objs []core.Object
	for _, o := range fos {
		objs = append(objs, o.Object)
	}
	return objs
}
