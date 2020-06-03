package nomostest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	"git-importer-raw-nomos-e2e.yaml",
	"monitor.yaml",
	"syncer.yaml",
}

// installConfigSync installs ConfigSync on the test cluster, and returns a
// callback for checking that the installation succeeded.
func installConfigSync(nt *NT) func() error {
	nt.T.Helper()
	tmpManifestsDir := filepath.Join(nt.TmpDir, manifests)

	objs := installationManifests(nt, tmpManifestsDir)

	for _, o := range objs {
		err := nt.Create(o)
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	return func() error {
		return Retry(60*time.Second, func() error {
			// TODO(willbeason): Ensure git-importer comes up as well.
			//  For now it isn't guaranteed to come up since the repository is empty.
			err := nt.Validate("monitor", configmanagement.ControllerNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
			if err != nil {
				return err
			}
			return nt.Validate("syncer", configmanagement.ControllerNamespace,
				&appsv1.Deployment{}, isAvailableDeployment)
		})
	}
}

func copyFile(src, dst string) error {
	bytes, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, bytes, os.ModePerm)
}

// installationManifests generates the ConfigSync installation YAML and copies
// it to the test's temporary directory.
func installationManifests(nt *NT, tmpManifestsDir string) []core.Object {
	nt.T.Helper()

	manifestFiles, err := ioutil.ReadDir(manifestsDir)
	if err != nil {
		nt.T.Fatal(err)
	}
	err = os.MkdirAll(tmpManifestsDir, os.ModePerm)
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

		err = ioutil.WriteFile(filepath.Join(tmpManifestsDir, template), []byte(replaced), os.ModePerm)
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
	hasGitSyncMap := false
	hasGitImporterDeployment := false
	for _, o := range fos {
		if o.GroupVersionKind() == kinds.ConfigMap() && o.GetName() == "git-sync" {
			hasGitSyncMap = true
			rmSSHFromConfigMap(nt, o)
		}
		if o.GroupVersionKind() == kinds.Deployment() && o.GetName() == "git-importer" {
			hasGitImporterDeployment = true
			rmSSHFromGitImporter(nt, o)
		}
		objs = append(objs, o.Object)
	}
	if !hasGitSyncMap {
		nt.T.Fatal("missing git-sync ConfigMap")
	}
	if !hasGitImporterDeployment {
		nt.T.Fatal("missing git-importer Deployment")
	}
	return objs
}

func rmSSHFromConfigMap(nt *NT, o ast.FileObject) {
	u, ok := o.Object.(*unstructured.Unstructured)
	if !ok {
		nt.T.Fatal(WrongTypeErr(o.Object, &unstructured.Unstructured{}))
	}
	err := unstructured.SetNestedField(u.Object,
		"false", "data", "GIT_SYNC_SSH")
	if err != nil {
		nt.T.Fatal(err)
	}
}

func rmSSHFromGitImporter(nt *NT, o ast.FileObject) {
	u, ok := o.Object.(*unstructured.Unstructured)
	if !ok {
		nt.T.Fatal(WrongTypeErr(o.Object, &unstructured.Unstructured{}))
	}
	containers, _, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "containers")
	if err != nil {
		nt.T.Fatal(err)
	}
	gitSyncContainer, ok := containers[2].(map[string]interface{})
	if !ok {
		nt.T.Fatal(WrongTypeErr(containers[2], make(map[string]interface{})))
	}
	volumeMounts, _, err := unstructured.NestedSlice(gitSyncContainer, "volumeMounts")
	if err != nil {
		nt.T.Fatal(err)
	}
	gitSyncContainer["volumeMounts"] = volumeMounts[:1]
	containers[2] = gitSyncContainer
	err = unstructured.SetNestedSlice(u.Object, containers, "spec", "template", "spec", "containers")
	if err != nil {
		nt.T.Fatal(err)
	}

	volumes, _, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "volumes")
	if err != nil {
		nt.T.Fatal(err)
	}
	volumes = volumes[:1]
	err = unstructured.SetNestedSlice(u.Object, volumes, "spec", "template", "spec", "volumes")
	if err != nil {
		nt.T.Fatal(err)
	}
}
