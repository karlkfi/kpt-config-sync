/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package kubectl contains the commands that we send to the kubectl binary.
package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/process/exec"
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	// nomosClusterNameConfigMap is the name of the configmap that contains
	// the name of the nomos cluster.
	nomosClusterNameConfigMap = "cluster-name"

	// nomosClusterNameKey is the name of the key in the above configmap that
	// contains the cluster name.
	nomosClusterNameKey = "CLUSTER_NAME"

	// nomosNamespace is the namespace in which Nomos runs by default.
	nomosNamespace = "nomos-system"
)

var (
	kubectlCmd = exec.RequireProgram("kubectl")

	// useMetaClient can be set to false by tests that don't need the meta
	// client.  Such tests  would fail in environments that need CGO, since meta
	// client is not CGO-clean.
	useMetaClient = true
)

// Context contains the runtime context for interacting with the Kubernetes client
// and server.
type Context struct {
	ctx    context.Context
	client meta.Interface
}

// NewWithClient creates a new kubernetes context from a predefined client
func NewWithClient(ctx context.Context, client meta.Interface) *Context {
	return &Context{ctx, client}
}

// New creates a new kubernetes context.
func New(ctx context.Context) *Context {
	var client *meta.Client
	if useMetaClient {
		// Use of the metaclient can be disabled in tests.  In that case, client
		// will be nil, but in the "regular" case, go initialize it.
		restConfig, err := restconfig.NewKubectlConfig()
		if err != nil {
			panic(errors.Wrapf(err, "failed to get restconfig"))
		}
		client = meta.NewForConfigOrDie(restConfig)
	}
	return &Context{ctx, client}
}

// Kubectl will execute a kubectl command.
func (t *Context) Kubectl(args ...string) (stdout, stderr string, err error) {
	actualArgs := append([]string{kubectlCmd}, args...)
	stdout, stderr, err = run(t.ctx, actualArgs)
	if glog.V(9) {
		glog.V(9).Infof("stdout: %v", stdout)
		glog.V(9).Infof("stderr: %v", stderr)
		glog.V(9).Infof("err: %v", err)
	}
	return // naked
}

// Apply runs kubectl apply -f on a given path.
func (t *Context) Apply(path string) error {
	if _, _, err := t.Kubectl("apply", "-f", path); err != nil {
		return errors.Wrapf(err, "while applying to path: %q", path)
	}
	return nil
}

// DeleteSecret deletes a secret from Kubernetes.
func (t *Context) DeleteSecret(name, namespace string) error {
	if err := t.Kubernetes().CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "delete secret name=%q, namespace=%q", name, namespace)
	}
	return nil
}

// CreateConfigMap creates a config map based on the passed-in values.
func (t *Context) CreateConfigMap(name, namespace string, data map[string]string) error {
	var cfg core.ConfigMap
	cfg.Data = data
	cfg.ObjectMeta.Name = name
	_, err := t.Kubernetes().CoreV1().ConfigMaps(namespace).Create(&cfg)
	return err
}

// DeleteConfigMap deletes a configmap from Kubernetes.  Fails if the configmap is not found,
// but returns error that can be tested with errors.IsNotFound(...).
func (t *Context) DeleteConfigMap(name, namespace string) error {
	err := t.Kubernetes().CoreV1().ConfigMaps(namespace).Delete(name, &metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		// Preserve the "is not foundness" of the error.
		return err
	}
	if err != nil {
		return errors.Wrapf(err, "delete configmap name=%q, namespace=%q", name, namespace)
	}
	return nil
}

// DeleteValidatingWebhookConfiguration deletes a validatingwebhookconfiguration from Kubernetes.
func (t *Context) DeleteValidatingWebhookConfiguration(name string) error {
	if err := t.Kubernetes().AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "delete validatingwebhookconfiguration name=%q", name)
	}
	return nil
}

// CreateSecretGenericFromFile creates a secret generic from the provided filenames
//
// Equivalent to:
// kubectl create secret generic -n=[namespace] --from-file=[filename0] --from-file=[filename1]...
func (t *Context) CreateSecretGenericFromFile(name, namespace string, filenames ...string) error {
	args := []string{
		"create", "secret", "generic", fmt.Sprintf("-n=%v", namespace), name,
	}
	for _, fn := range filenames {
		args = append(args, fmt.Sprintf("--from-file=%q", filepath.Clean(fn)))
	}
	if _, _, err := t.Kubectl(args...); err != nil {
		return errors.Wrapf(err, "create secret generic name=%q, namespace=%q", name, namespace)
	}
	return nil
}

// CreateConfigmapFromLiterals creates a Kubernetes configmap from key-value
// pairs represented as strings like: "KEY1=VALUE1", "KEY2=VALUE2", etc.
//
// Equivalent to:
//   kubectl create configmap name -n=space --from-literal=KEY1=VALUE1 ...
func (t *Context) CreateConfigmapFromLiterals(name, namespace string, literals ...string) error {
	args := []string{
		"create", "configmap", name, fmt.Sprintf("-n=%v", namespace),
	}
	for _, l := range literals {
		args = append(args, fmt.Sprintf("--from-literal=%v", l))
	}
	if _, _, err := t.Kubectl(args...); err != nil {
		return errors.Wrapf(err, "create configmap name=%q, namespace=%q", name, namespace)
	}
	return nil
}

// clusterAdminRoleBingingName returns the name of the ClusterRoleBinding of the provided
// user to the cluster-admin role, or what that name should be if the binding were to exist.
func clusterAdminRoleBindingName(user string) string {
	return fmt.Sprintf("%v-cluster-admin-binding", user)
}

// AddClusterAdmin adds user as a cluster admin.  This is only useful on clusters
// that require such a change.  For example GKE.
func (t *Context) AddClusterAdmin(user string) error {
	// Ensure that at the beginning there is no permission for the current user.
	if err := t.RemoveClusterAdmin(user); err != nil {
		return errors.Wrapf(err, "while trying to clean up cluster admin for user")
	}
	name := clusterAdminRoleBindingName(user)
	cr := &v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []v1.Subject{{
			Kind: v1.UserKind,
			Name: user,
		}},
		RoleRef: v1.RoleRef{
			Kind: "ClusterRole",
			Name: "cluster-admin",
		},
	}
	if _, err := t.Kubernetes().RbacV1().ClusterRoleBindings().Create(cr); err != nil {
		return errors.Wrapf(err, "making admin: %q", user)
	}
	return nil
}

// RemoveClusterAdmin removes the user from the cluster admin role.  This is only
// useful on GKE, and does nothing on other platforms.
func (t *Context) RemoveClusterAdmin(user string) error {
	name := clusterAdminRoleBindingName(user)
	if err := t.Kubernetes().RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "unmaking admin: %q", user)
	}
	return nil
}

// GetClusterVersion obtains the semantic version information from the cluster in the
// current context.
func (t *Context) GetClusterVersion() (semver.Version, error) {
	serv, err := t.Kubernetes().Discovery().ServerVersion()
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "while getting cluster version")
	}
	// GitVersion is of the form "v1.9.2-something". Strip off the "v"" and return.
	semv, err := semver.Parse(serv.GitVersion[1:])
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "while parsing version")
	}
	return semv, nil
}

// ClusterList encapsulates a list of clusters
type ClusterList struct {
	// Clusters is the list of clusters available to the user, keyed by the
	// name of the context.
	Clusters map[string]string

	// Current the name of the context marked as "current" in the Clusters.
	Current string
}

// SetContext sets the cluster context to the context with the given name.  The
// named context must exist.
func (t *Context) SetContext(name string) error {
	_, _, err := t.Kubectl("config", "use-context", name)
	if err != nil {
		return errors.Wrapf(err, "while setting context to: %q", name)
	}
	return nil
}

// LocalClusters gets the list of available clusters from the local client
// configuration.
func LocalClusters() (ClusterList, error) {
	clientConfig, err := restconfig.NewClientConfig()
	if err != nil {
		return ClusterList{}, errors.Wrapf(err, "Clusters()")
	}
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return ClusterList{}, errors.Wrapf(err, "RawConfig()")
	}
	return Clusters(rawConfig), nil
}

// Clusters gets the list of available clusters from the supplied client
// configuration.
func Clusters(c clientcmdapi.Config) ClusterList {
	cl := ClusterList{
		Current:  c.CurrentContext,
		Clusters: map[string]string{},
	}
	cl.Current = c.CurrentContext
	for name, context := range c.Contexts {
		cl.Clusters[name] = context.Cluster
	}
	return cl
}

// run runs the command with the given context and arguments.  Returns the contents of
// stdout and stderr, and error if any.
func run(ctx context.Context, args []string) (stdout, stderr string, err error) {
	outbuf := bytes.NewBuffer(nil)
	errbuf := bytes.NewBuffer(nil)
	e := exec.NewRedirected(os.Stdin, outbuf, errbuf).Run(ctx, args...)
	if e != nil {
		// Embed stderr into the error report in case of failure, that's
		// where the error message is most of the time.
		e = errors.Wrapf(e, "stderr: %v", errbuf.String())
	}
	return outbuf.String(), errbuf.String(), e
}

// waitForDeployment watches for the deployment of the provided name to become available and
// returns when it becomes so. If the provided deadline is passed or another error is encountered,
// an error is returned and the deployment is not presumed to be available.
func (t *Context) waitForDeployment(deadline time.Time, namespace string, name string) error {
	glog.V(2).Infof("Waiting for deployment %s to become available...", name)
	deployment, err := t.Kubernetes().ExtensionsV1beta1().Deployments(
		namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error getting deployment %s:%s", namespace, name)
	}
	time.Sleep(time.Duration(deployment.Spec.MinReadySeconds) * time.Second)
	for time.Now().Before(deadline) {
		deployment, err = t.Kubernetes().ExtensionsV1beta1().Deployments(
			namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "Error getting deployment %s:%s", namespace, name)
		}
		glog.V(5).Infof(
			"Deployment %s replicas %d, available %d", name, deployment.Spec.Replicas, deployment.Status.AvailableReplicas)
		if deployment.Status.UnavailableReplicas == 0 {
			glog.V(1).Infof("Deployment %s available", name)
			return nil
		}
		time.Sleep(time.Millisecond * 250)
	}
	return errors.Errorf("Deployment %s:%s failed to become available before deadline", namespace, name)
}

// WaitForDeployments waits for deployments to be available, or returns error in
// case of failure.
func (t *Context) WaitForDeployments(timeout time.Duration, ns string, deployments ...string) error {
	deadline := time.Now().Add(timeout)
	for _, d := range deployments {
		err := t.waitForDeployment(deadline, ns, d)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteDeployment deletes a deployment in the given namespace.  No effect if
// the deployment isn't already running.
func (t *Context) DeleteDeployment(name, namespace string) error {
	if err := t.Kubernetes().AppsV1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting deployment: %v:%v", namespace, name)
	}
	return nil
}

// DeleteNamespace deletes the supplied namespace.
func (t *Context) DeleteNamespace(name string) error {
	if err := t.Kubernetes().CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting namespace: %q", name)
	}
	return nil
}

// WaitForNamespaceDeleted waits until the named namespace is no longer there,
// or a timeout occurs.
func (t *Context) WaitForNamespaceDeleted(namespace string) error {
	return wait.Poll(10*time.Second, 3*time.Minute, func() (done bool, e error) {
		ns, err := t.Kubernetes().CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "while listing namespace: %q", namespace)
		}
		for _, n := range ns.Items {
			if n.ObjectMeta.Name == namespace {
				glog.V(5).Infof("namespace %q still present, continuing poll", namespace)
				return false, nil
			}
		}
		glog.V(9).Infof("namespace %q is now gone, ending poll", namespace)
		return true, nil
	})
}

// DeleteClusterrolebinding deletes a cluster role binding by the given name.
func (t *Context) DeleteClusterrolebinding(name string) error {
	if err := t.Kubernetes().RbacV1().ClusterRoleBindings().Delete(
		name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting clusterrolebinding %q", name)
	}
	return nil
}

// DeleteClusterrole deletes a cluster role by the given name.
func (t *Context) DeleteClusterrole(name string) error {
	if err := t.Kubernetes().RbacV1().ClusterRoles().Delete(
		name, &metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting clusterrole %q", name)
	}
	return nil
}

// DeleteDeprecatedCRD deletes the named CRD if its version matches version.
// This should be used specifically to remove CRDs whose version changed. Removing CRDs
// in general is dangerous as it deletes all the resources of that CRD.
// Kubernetes 1.13 has support for properly versioning CRDs at which point this function can
// be deprecated.
func (t *Context) DeleteDeprecatedCRD(name, version string) error {
	crd, err := t.APIExtensions().ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err == nil && crd.Spec.Group == "nomos.dev" && crd.Spec.Version == version {
		glog.Infof("Removing deprecated CRD version %s/%s", name, version)
		err2 := t.APIExtensions().ApiextensionsV1beta1().CustomResourceDefinitions().Delete(name, &metav1.DeleteOptions{})
		if err2 != nil {
			return err2
		}
	}

	return nil
}

type nomosError struct {
	emptyName bool
}

// Error implements error.
func (ne nomosError) Error() string {
	switch {
	case ne.emptyName:
		return "empty name"
	default:
		return "unknown nomos error"

	}
}

// IsNomosEmptyName returns true if the error is due to the empty name.
func IsNomosEmptyName(err error) bool {
	ne, ok := err.(nomosError)
	if !ok {
		return false
	}
	return ne.emptyName
}

// CreateClusterName creates a Nomos object for the specified clusterName.  An already existing
// cluster name is overwritten with a new one.
func (t *Context) CreateClusterName(clusterName string) error {
	glog.V(6).Infof("CreateClusterName(%q): ENTER", clusterName)
	defer glog.V(6).Infof("CreateClusterName(%q): EXIT", clusterName)
	// Delete cluster name regardless of whether cluster is now named or not.
	if err := t.DeleteClusterName(); err != nil {
		return errors.Wrapf(err, "while removing cluster name: %q", clusterName)
	}
	if clusterName == "" {
		return nomosError{emptyName: true}
	}
	data := map[string]string{nomosClusterNameKey: clusterName}
	if err := t.CreateConfigMap(nomosClusterNameConfigMap, nomosNamespace, data); err != nil {
		return errors.Wrapf(err, "while creating Nomos for cluster %q", clusterName)
	}
	return nil
}

// GetClusterName gets the nomos cluster name.
func (t *Context) GetClusterName() (string, error) {
	c, err := t.Kubernetes().CoreV1().ConfigMaps(nomosNamespace).Get(nomosClusterNameConfigMap, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	fmt.Printf("GetClusterName: %v", c)

	return c.Data["CLUSTER_NAME"], err
}

// DeleteClusterName deletes the cluster name object.  If the object does not
// already exist, no change is made.
func (t *Context) DeleteClusterName() error {
	err := t.DeleteConfigMap(nomosClusterNameConfigMap, nomosNamespace)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// Kubernetes returns the underlying Kubernetes client.
func (t *Context) Kubernetes() kubernetes.Interface {
	return t.client.Kubernetes()
}

// PolicyHierarchy returns the policyhierarchy client interface
func (t *Context) PolicyHierarchy() apis.Interface {
	return t.client.PolicyHierarchy()
}

// APIExtensions returns the apiextensions client interface
func (t *Context) APIExtensions() apiextensions.Interface {
	return t.client.APIExtensions()
}
