package nomostest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	acmeDir       = "acme"
	manifests     = "manifests"
	testResources = "test-resources"

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
	baseDir          = filepath.FromSlash("../..")
	manifestsDir     = filepath.Join(baseDir, manifests)
	testResourcesDir = filepath.Join(manifestsDir, testResources)
	templateDir      = filepath.Join(manifestsDir, "templates")

	monoConfigMaps  = filepath.Join(baseDir, "e2e", "raw-nomos", manifests, monoConfigMapsName)
	multiConfigMaps = filepath.Join(baseDir, "e2e", "raw-nomos", manifests, multiConfigMapsName)

	// clusterRoleName is the ClusterRole used by Namespace Reconciler.
	clusterRoleName = fmt.Sprintf("%s:%s", configsync.GroupName, "ns-reconciler")
	// RepoSyncFileName specifies the filename containing RepoSync.
	RepoSyncFileName = "repo-sync.yaml"

	templates = []string{
		"git-importer.yaml",
		"monitor.yaml",
		"reconciler-manager.yaml",
		"reconciler-manager-configmap.yaml",
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

// filesystemPollingPeriod specifies filesystem polling period as time.Duration
var filesystemPollingPeriod time.Duration

// IsReconcilerManagerConfigMap returns true if passed obj is the
// reconciler-manager ConfigMap reconciler-manager-cm in config-management namespace.
var IsReconcilerManagerConfigMap = func(obj core.Object) bool {
	return obj.GetName() == "reconciler-manager-cm" &&
		obj.GetNamespace() == "config-management-system" &&
		obj.GroupVersionKind() == kinds.ConfigMap()
}

// gitRepo returns fully qualified git repo name hosted in test git server.
func gitRepo(repoName string) string {
	return fmt.Sprintf("git@test-git-server.config-management-system-test:/git-server/repos/%s", repoName)
}

// installConfigSync installs ConfigSync on the test cluster, and returns a
// callback for checking that the installation succeeded.
func installConfigSync(nt *NT, nomos ntopts.Nomos) func(*NT) error {
	nt.T.Helper()
	tmpManifestsDir := filepath.Join(nt.TmpDir, manifests)

	objs := installationManifests(nt, tmpManifestsDir)
	if nomos.MultiRepo {
		filesystemPollingPeriod = nt.FilesystemPollingPeriod
		objs = multiRepoObjects(nt.T, objs, setReconcilerDebugMode, setReconcilerFilesystemPollingPeriod)
	} else {
		objs = monoRepoObjects(objs)
	}
	objs = convertObjects(nt, objs)

	for _, o := range objs {
		if o.GroupVersionKind().GroupKind() == kinds.ConfigMap().GroupKind() && o.GetName() == controllers.SourceFormat {
			cm := o.(*corev1.ConfigMap)
			cm.Data[filesystem.SourceFormatKey] = string(nomos.SourceFormat)
		}

		err := nt.Create(o)
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	if nomos.MultiRepo {
		return validateMultiRepoDeployments
	}
	return validateMonoRepoDeployments
}

// convertObjects converts objects to their literal types. We can do this as
// we should have all required types in the scheme anyway. This keeps us from
// having to do ugly Unstructured operations.
func convertObjects(nt *NT, objs []core.Object) []core.Object {
	result := make([]core.Object, len(objs))
	for i, obj := range objs {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			// Somehow already converted, or added manually.
			result[i] = obj
		}

		o, err := nt.scheme.New(u.GroupVersionKind())
		if err != nil {
			nt.T.Fatalf("installed type %v not in Scheme: %v", u.GroupVersionKind(), err)
		}

		jsn, err := u.MarshalJSON()
		if err != nil {
			nt.T.Fatalf("marshalling object into JSON: %v", err)
		}

		err = json.Unmarshal(jsn, o)
		if err != nil {
			nt.T.Fatalf("unmarshalling JSON into object: %v", err)
		}
		newObj, ok := o.(core.Object)
		if !ok {
			nt.T.Fatalf("trying to install non-object type %v", u.GroupVersionKind())
		}
		result[i] = newObj
	}
	return result
}

func copyFile(src, dst string) error {
	bytes, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, bytes, fileMode)
}

func copyDirContents(src, dest string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		// Explicitly not recursive
		if f.IsDir() {
			continue
		}

		from := filepath.Join(src, f.Name())
		to := filepath.Join(dest, f.Name())
		err := copyFile(from, to)
		if err != nil {
			return err
		}
	}
	return nil
}

// installationManifests generates the ConfigSync installation YAML and copies
// it to the test's temporary directory.
func installationManifests(nt *NT, tmpManifestsDir string) []core.Object {
	nt.T.Helper()
	err := os.MkdirAll(tmpManifestsDir, fileMode)
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("copying test-only-resources")
	err = copyDirContents(testResourcesDir, tmpManifestsDir)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.T.Log("copying manifests, not including manifests/templates/")
	err = copyDirContents(manifestsDir, tmpManifestsDir)
	if err != nil {
		nt.T.Fatal(err)
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
			imgName = *e2e.ImagePrefix + "/reconciler-manager:latest"
		case "reconciler-manager-configmap.yaml":
			// For the reconciler deployment template, we want the latest image for the reconciler.
			imgName = *e2e.ImagePrefix + "/reconciler:latest"
		default:
			// For any other template, we want the latest image for the nomos binary (mono-repo).
			imgName = *e2e.ImagePrefix + "/nomos:latest"
		}

		replaced := strings.ReplaceAll(string(bytes), "IMAGE_NAME", imgName)

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
	filePaths := filesystem.FilePaths{
		RootDir: readPath,
		Files:   paths,
	}
	fos, err := reader.Read(filePaths)
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

func multiRepoObjects(t *testing.T, objects []core.Object, opts ...func(t *testing.T, obj core.Object)) []core.Object {
	var filtered []core.Object
	found := false
	for _, obj := range objects {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			t.Fatalf("parsed multi repo object: %q was not Unstructured %v", u.GetName(), u)
		}
		if IsReconcilerManagerConfigMap(obj) {
			// Mark that we've found the ReconcilerManager ConfigMap.
			// This way we know we've enabled debug mode.
			found = true
		}
		for _, opt := range opts {
			opt(t, obj)
		}
		if multiObjects[obj.GetName()] || sharedObjects[obj.GetName()] {
			filtered = append(filtered, obj)
		}
	}
	if !found {
		t.Fatal("Did not find Reconciler Manager ConfigMap")
	}
	return filtered
}

func validateMonoRepoDeployments(nt *NT) error {
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

func validateMultiRepoDeployments(nt *NT) error {
	// Create a RootSync to initialize the root reconciler.
	rs := fake.RootSyncObject()
	rs.Spec.SourceFormat = string(nt.Root.Format)
	rs.Spec.Git = v1alpha1.Git{
		Repo:      gitRepo(rootRepo),
		Branch:    MainBranch,
		Dir:       acmeDir,
		Auth:      "ssh",
		SecretRef: v1alpha1.SecretReference{Name: "git-creds"},
	}
	if err := nt.Create(rs); err != nil {
		nt.T.Fatal(err)
	}

	took, err := Retry(60*time.Second, func() error {
		// USE CAUTION WHEN ADDING THINGS HERE.
		// This is not a place for test code. The only things that belong here are
		// test preconditions, i.e. things that mean *every* test that uses
		// multi-repo functionality will fail, and none of these tests should
		// continue.
		err := nt.Validate("reconciler-manager", configmanagement.ControllerNamespace,
			&appsv1.Deployment{}, isAvailableDeployment)
		if err != nil {
			return err
		}
		return nt.Validate("root-reconciler", configmanagement.ControllerNamespace,
			&appsv1.Deployment{}, isAvailableDeployment)
	})
	if err != nil {
		return err
	}
	nt.T.Logf("took %v to wait for reconciler-manager and root-reconciler", took)
	return nil
}

func setupRepoSync(nt *NT, ns string) {
	// create RepoSync to initialize the Namespace reconciler.
	rs := repoSyncObject(ns)
	if err := nt.Create(rs); err != nil {
		nt.T.Fatal(err)
	}
}

func waitForRepoReconciler(nt *NT, ns string) error {
	took, err := Retry(60*time.Second, func() error {
		// USE CAUTION WHEN ADDING THINGS HERE.
		// This is not a place for test code. The only things that belong here are
		// test preconditions, i.e. things that mean *every* test that uses
		// multi-repo functionality will fail, and none of these tests should
		// continue.
		return nt.Validate(fmt.Sprintf("ns-reconciler-%s", ns), configmanagement.ControllerNamespace,
			&appsv1.Deployment{}, isAvailableDeployment)
	})
	if err != nil {
		return err
	}
	nt.T.Logf("took %v to wait for ns-reconciler", took)

	return nil
}

func repoSyncClusterRole() *rbacv1.ClusterRole {
	cr := fake.ClusterRoleObject(core.Name(clusterRoleName))
	cr.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{rbacv1.APIGroupAll},
			Resources: []string{rbacv1.ResourceAll},
			Verbs:     []string{rbacv1.VerbAll},
		},
	}
	return cr
}

// repoSyncRoleBinding returns rolebinding that grants service account
// permission to manage resources in the namespace.
func repoSyncRoleBinding(ns string) *rbacv1.RoleBinding {
	rb := fake.RoleBindingObject(core.Name("syncs"), core.Namespace(ns))
	sb := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      fmt.Sprintf("ns-reconciler-%s", ns),
			Namespace: configmanagement.ControllerNamespace,
		},
	}
	rf := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     clusterRoleName,
	}
	rb.Subjects = sb
	rb.RoleRef = rf
	return rb
}

func setupRepoSyncRoleBinding(nt *NT, ns string) error {
	if err := nt.Create(repoSyncRoleBinding(ns)); err != nil {
		nt.T.Fatal(err)
	}

	// Validate rolebinding 'syncs' is present.
	return nt.Validate("syncs", ns, &rbacv1.RoleBinding{})
}

// setReconcilerDebugMode ensures the Reconciler deployments are run in debug mode.
func setReconcilerDebugMode(t *testing.T, obj core.Object) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("parsed Reconciler Manager ConfigMap was not Unstructured %v", u)
	}

	if !IsReconcilerManagerConfigMap(u) {
		return
	}
	deploymentYaml, found, err := unstructured.NestedString(u.Object, "data", "deployment.yaml")
	if !found {
		t.Fatal("Reconciler Manager ConfigMap has no deployment.yaml entry")
	}
	if err != nil {
		t.Fatalf("Getting deployment.yaml entry from Reconciler Manager ConfigMap: %v", err)
	}

	// The Deployment YAML for Reconciler deployments is a raw YAML string embedded
	// in the ConfigMap. Unmarshalling/marshalling is likely to lead to errors, so
	// this modifies the YAML string directly by finding the line we want to insert
	// the debug flag after, and then inserting the line we want to add.
	lines := strings.Split(deploymentYaml, "\n")
	found = false
	for i, line := range lines {
		// We want to set the debug flag immediately after setting the git-dir flag.
		if strings.Contains(line, "- \"--git-dir=/repo/root/rev\"") {
			// Standard Go "insert into slice" idiom.
			lines = append(lines, "")
			copy(lines[i+2:], lines[i+1:])
			// Prefix of 8 spaces as the run arguments are indented 8 spaces relative
			// to the embedded YAML string. The embedded YAML is indented 3 spaces,
			// so this is equivalent to indenting 11 spaces in the original file:
			// manifests/templates/reconciler-manager-configmap.yaml.
			lines[i+1] = "        - \"--debug\""
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Unable to set debug mode for reconciler")
	}

	err = unstructured.SetNestedField(u.Object,
		strings.Join(lines, "\n"), "data", "deployment.yaml")
	if err != nil {
		t.Fatalf("Setting deployment.yaml: %v", err)
	}
	t.Log("Set deployment.yaml")
}

// setReconcilerFilesystemPollingPeriod update Reconciler Manager configmap
// reconciler-manager-cm with filesystem polling period to override the default.
func setReconcilerFilesystemPollingPeriod(t *testing.T, obj core.Object) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("parsed Reconciler Manager ConfigMap was not Unstructured %v", u)
	}
	if !IsReconcilerManagerConfigMap(u) {
		return
	}
	err := unstructured.SetNestedField(u.Object,
		filesystemPollingPeriod.String(), "data", controllers.FilesystemPollingPeriod)
	if err != nil {
		t.Fatalf("Setting filesystem polling period: %v", err)
	}
	t.Log("Set filesystem polling period")
}

func setupDelegatedControl(nt *NT, opts ntopts.New) {
	// Just create one RepoSync ClusterRole, even if there are no Namespace repos.
	if err := nt.Create(repoSyncClusterRole()); err != nil {
		nt.T.Fatal(err)
	}

	for ns := range opts.MultiRepo.NamespaceRepos {
		nt.NamespaceRepos[ns] = ns

		// create namespace for namespace reconciler.
		err := nt.Create(fake.NamespaceObject(ns))
		if err != nil {
			nt.T.Fatal(err)
		}

		// create secret for the namespace reconciler.
		CreateNamespaceSecret(nt, ns)

		if err := setupRepoSyncRoleBinding(nt, ns); err != nil {
			nt.T.Fatal(err)
		}

		setupRepoSync(nt, ns)
	}

	for ns := range opts.MultiRepo.NamespaceRepos {
		if err := waitForRepoReconciler(nt, ns); err != nil {
			nt.T.Fatal(err)
		}
	}
}

// StructuredNSPath returns structured path with namespace and resourcename in repo.
func StructuredNSPath(namespace, resourceName string) string {
	return fmt.Sprintf("acme/namespaces/%s/%s", namespace, resourceName)
}

func repoSyncObject(ns string) *v1alpha1.RepoSync {
	rs := fake.RepoSyncObject(core.Namespace(ns))
	rs.Spec.Git = v1alpha1.Git{
		Repo:   gitRepo(ns),
		Branch: MainBranch,
		Dir:    acmeDir,
		Auth:   "ssh",
		SecretRef: v1alpha1.SecretReference{
			Name: "ssh-key",
		},
	}
	return rs
}

func setupCentralizedControl(nt *NT, opts ntopts.New) {
	for ns := range opts.MultiRepo.NamespaceRepos {
		nt.Root.Add(StructuredNSPath(ns, "ns.yaml"), fake.NamespaceObject(ns))
		nt.Root.Add("acme/cluster/cr.yaml", repoSyncClusterRole())
		nt.Root.Add(StructuredNSPath(ns, "rb.yaml"), repoSyncRoleBinding(ns))

		rs := repoSyncObject(ns)
		nt.Root.Add(StructuredNSPath(ns, RepoSyncFileName), rs)

		nt.Root.CommitAndPush("Adding namespace,clusterrole, rolebinding and RepoSync")
		// This waits for the Namespace to be created.
		nt.WaitForRepoSyncs()

		// Now that the Namespace exists, create the secret inside it, and ensure
		// its RepoSync reports everything is synced.
		CreateNamespaceSecret(nt, ns)
		nt.NamespaceRepos[ns] = ns
		nt.WaitForRepoSyncs()

		err := nt.Validate(rs.Name, ns, &v1alpha1.RepoSync{})
		if err != nil {
			nt.T.Fatal(err)
		}
	}
}
