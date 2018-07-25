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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/policyhierarchy"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/process/exec"
	"github.com/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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

// KubectlOrDie will execute a kubectl command and panic if the script fails.
func (t *Context) KubectlOrDie(args ...string) (stdout, stderr string) {
	stdout, stderr, err := t.Kubectl(args...)
	if err != nil {
		panic(errors.Errorf("Command %s failed, stdout: %s stderr: %s", strings.Join(args, " "), stdout, stderr))
	}
	return stdout, stderr
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
	if err := t.Kubernetes().CoreV1().Secrets(namespace).Delete(name, &meta_v1.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "delete secret name=%q, namespace=%q", name, namespace)
	}
	return nil
}

// DeleteConfigMap deletes a configmap from Kubernetes.
func (t *Context) DeleteConfigMap(name, namespace string) error {
	if err := t.Kubernetes().CoreV1().ConfigMaps(namespace).Delete(name, &meta_v1.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "delete configmap name=%q, namespace=%q", name, namespace)
	}
	return nil
}

// DeleteValidatingWebhookConfiguration deletes a validatingwebhookconfiguration from Kubernetes.
func (t *Context) DeleteValidatingWebhookConfiguration(name string) error {
	if _, _, err := t.Kubectl("delete", "validatingwebhookconfiguration", name, "--ignore-not-found"); err != nil {
		return errors.Wrapf(err, "delete validatingwebhookconfiguration name=%q")
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

// AddClusterAdmin adds user as a cluster admin.  This is only useful on clusters
// that require such a change.  For example GKE.
func (t *Context) AddClusterAdmin(user string) error {
	// Ensure that at the beginning there is no permission for the current user.
	if err := t.RemoveClusterAdmin(user); err != nil {
		return errors.Wrapf(err, "while trying to clean up cluster admin for user")
	}
	// TODO(filmil): 'user' here comes from user-supplied configuration.  Should
	// it be sanitized, or is placement in 'args' enough?
	args := []string{
		"create", "clusterrolebinding",
		fmt.Sprintf("%v-cluster-admin-binding", user),
		"--clusterrole=cluster-admin",
		fmt.Sprintf("--user=%v", user),
	}
	if _, _, err := t.Kubectl(args...); err != nil {
		return errors.Wrapf(err, "making admin: %q", user)
	}
	return nil
}

// RemoveClusterAdmin removes the user from the cluster admin role.  This is only
// useful on GKE, and does nothing on other platforms.
func (t *Context) RemoveClusterAdmin(user string) error {
	args := []string{
		"delete", "clusterrolebinding",
		fmt.Sprintf("%v-cluster-admin-binding", user),
		"--ignore-not-found",
	}
	if _, _, err := t.Kubectl(args...); err != nil {
		return errors.Wrapf(err, "unmaking admin: %q", user)
	}
	return nil
}

// versionInfo is a partial parsed output of the "kubectl version" command.
type versionInfo struct {
	GitVersion string `json:"gitVersion"`
}
type versionOutput struct {
	ClientVersion versionInfo `json:"clientVersion"`
	ServerVersion versionInfo `json:"serverVersion"`
}

// GetClusterVersion obtains the semantic version information from the cluster in the
// current context.
func (t *Context) GetClusterVersion() (semver.Version, error) {
	stdout, stderr, err := t.Kubectl("version", "-o", "json")
	if glog.V(8) {
		glog.Infof("stdout: %v\nstderr:%v", stdout, stderr)
	}
	if stderr != "" {
		glog.Warningf("GetClusterVersion(): nonempty stderr: %v", stderr)
	}
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "while getting cluster version")
	}
	var vs versionOutput
	err = json.Unmarshal([]byte(stdout), &vs)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "while unmarshalling")
	}
	glog.Warningf("vs: %+v", vs)
	// GitVersion is of the form "v1.9.2-something"
	version := vs.ServerVersion.GitVersion[1:]
	v, err := semver.Parse(version)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "while parsing version")
	}
	return v, nil
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
		namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error getting deployment %s:%s", namespace, name)
	}
	time.Sleep(time.Duration(deployment.Spec.MinReadySeconds) * time.Second)
	for time.Now().Before(deadline) {
		deployment, err = t.Kubernetes().ExtensionsV1beta1().Deployments(
			namespace).Get(name, meta_v1.GetOptions{})
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
	if err := t.Kubernetes().AppsV1().Deployments(namespace).Delete(name, &meta_v1.DeleteOptions{}); err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting deployment: %v:%v", namespace, name)
	}
	return nil
}

// DeleteNamespace deletes the supplied namespace.
func (t *Context) DeleteNamespace(name string) error {
	if err := t.Kubernetes().CoreV1().Namespaces().Delete(name, &meta_v1.DeleteOptions{}); err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting namespace: %q", name)
	}
	return nil
}

// WaitForNamespaceDeleted waits until the named namespace is no longer there,
// or a timeout occurs.
func (t *Context) WaitForNamespaceDeleted(namespace string) error {
	return wait.Poll(10*time.Second, 3*time.Minute, func() (done bool, e error) {
		ns, err := t.Kubernetes().CoreV1().Namespaces().List(meta_v1.ListOptions{})
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
		name, &meta_v1.DeleteOptions{}); err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting clusterrolebinding %q", name)
	}
	return nil
}

// DeleteClusterrole deletes a cluster role by the given name.
func (t *Context) DeleteClusterrole(name string) error {
	if err := t.Kubernetes().RbacV1().ClusterRoles().Delete(
		name, &meta_v1.DeleteOptions{}); err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "while deleting clusterrole %q", name)
	}
	return nil
}

// Kubernetes returns the underlying Kubernetes client.
func (t *Context) Kubernetes() kubernetes.Interface {
	return t.client.Kubernetes()
}

// PolicyHierarchy returns the policyhierarchy client interface
func (t *Context) PolicyHierarchy() policyhierarchy.Interface {
	return t.client.PolicyHierarchy()
}
