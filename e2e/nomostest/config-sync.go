package nomostest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	manifests = "manifests"

	// e2e/raw-nomos/manifests/mono-repo-configmaps.yaml
	monoConfigMapsName = "mono-repo-configmaps.yaml"
	// e2e/raw-nomos/manifests/multi-repo-configmaps.yaml
	multiConfigMapsName = "multi-repo-configmaps.yaml"
)

var (
	// baseDir is the path to the Nomos repository root from test case files.
	//
	// All paths must be relative to the test file that is running. There is probably
	// a more elegant way to do this.
	baseDir      = filepath.FromSlash("../..")
	manifestsDir = filepath.Join(baseDir, manifests)
	templateDir  = filepath.Join(manifestsDir, "templates")

	monoConfigMaps  = filepath.Join(baseDir, "e2e", "raw-nomos", manifests, monoConfigMapsName)
	multiConfigMaps = filepath.Join(baseDir, "e2e", "raw-nomos", manifests, multiConfigMapsName)

	templates = []string{
		"git-importer.yaml",
		"monitor.yaml",
		"reconciler-manager.yaml",
		"reconciler-manager-deployment-configmap.yaml",
	}

	// monoObjects contains the names of all objects that are necessary to install
	// and run mono-repo Config Sync.
	monoObjects = map[string]bool{
		"configmanagement.gke.io:importer":         true,
		"configmanagement.gke.io:monitor":          true,
		"clusterconfigs.configmanagement.gke.io":   true,
		"cluster-name":                             true,
		"git-importer":                             true,
		"git-sync":                                 true,
		"hierarchyconfigs.configmanagement.gke.io": true,
		"importer":                                 true,
		"monitor":                                  true,
		"namespaceconfigs.configmanagement.gke.io": true,
		"repos.configmanagement.gke.io":            true,
		"source-format":                            true,
		"syncs.configmanagement.gke.io":            true,
	}
	// multiObjects contains the names of all objects that are necessary to
	// install and run multi-repo Config Sync.
	multiObjects = map[string]bool{
		"configsync.gke.io:reconciler-manager": true,
		"reconciler-manager":                   true,
		"reconciler-manager-cm":                true,
		"reposyncs.configsync.gke.io":          true,
		"rootsyncs.configsync.gke.io":          true,
	}
	// sharedObjects contains the names of all objects that are needed by both
	// mono-repo and multi-repo Config Sync.
	sharedObjects = map[string]bool{
		"clusters.clusterregistry.k8s.io":            true,
		"clusterselectors.configmanagement.gke.io":   true,
		"container-limits":                           true,
		"namespaceselectors.configmanagement.gke.io": true,
	}
	// ignoredObjects:
	// config-management-system, this namespace gets created elsewhere
)

// installConfigSync installs ConfigSync on the test cluster, and returns a
// callback for checking that the installation succeeded.
func installConfigSync(nt *NT, nomos ntopts.Nomos) func() error {
	nt.T.Helper()
	tmpManifestsDir := filepath.Join(nt.TmpDir, manifests)

	objs := installationManifests(nt, tmpManifestsDir)
	if nomos.MultiRepo {
		objs = multiRepoObjects(objs)
	} else {
		objs = monoRepoObjects(objs)
	}

	for _, o := range objs {
		if o.GroupVersionKind().GroupKind() == kinds.ConfigMap().GroupKind() && o.GetName() == controllers.SourceFormat {
			u := o.(*unstructured.Unstructured)
			data := u.Object["data"].(map[string]interface{})
			data[filesystem.SourceFormatKey] = string(nomos.SourceFormat)
		}
		err := nt.Create(o)
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	return func() error {
		took, err := Retry(60*time.Second, func() error {
			err := nt.Validate("monitor", configmanagement.ControllerNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
			if err != nil {
				return err
			}
			return nt.Validate("git-importer", configmanagement.ControllerNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
		})
		if err != nil {
			return err
		}
		nt.T.Logf("took %v to wait for monitor and git-importer", took)
		return nil
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
		nt.T.Log("copying manifests")
		// Explicitly not recursive since we want to treat the template files specially.
		if !f.IsDir() {
			from := filepath.Join(manifestsDir, f.Name())
			to := filepath.Join(tmpManifestsDir, f.Name())
			err = copyFile(from, to)
			if err != nil {
				nt.T.Fatal(err)
			}
		}
	}

	// Copy ConfigMaps
	err = copyFile(monoConfigMaps, filepath.Join(tmpManifestsDir, monoConfigMapsName))
	if err != nil {
		nt.T.Fatal(err)
	}
	err = copyFile(multiConfigMaps, filepath.Join(tmpManifestsDir, multiConfigMapsName))
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

		var imgName string
		switch template {
		case "reconciler-manager.yaml":
			// For the reconciler manager template, we want the latest image for the reconciler manager.
			imgName = "localhost:5000/reconciler-manager:latest"
		case "reconciler-manager-deployment-configmap.yaml":
			// For the reconciler deployment template, we want the latest image for the reconciler.
			imgName = "localhost:5000/reconciler:latest"
		default:
			// For any other template, we want the latest image for the nomos binary (mono-repo).
			imgName = "localhost:5000/nomos:latest"
		}

		replaced := strings.Replace(string(bytes), "IMAGE_NAME", imgName, -1)

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

func monoRepoObjects(objects []core.Object) []core.Object {
	var filtered []core.Object
	for _, obj := range objects {
		if monoObjects[obj.GetName()] || sharedObjects[obj.GetName()] {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}

func multiRepoObjects(objects []core.Object) []core.Object {
	var filtered []core.Object
	for _, obj := range objects {
		if multiObjects[obj.GetName()] || sharedObjects[obj.GetName()] {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}
