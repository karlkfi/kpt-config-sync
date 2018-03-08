/*
Copyright 2018 The Stolos Authors.
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
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/policyhierarchy"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/google/stolos/pkg/toolkit/exec"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	client *meta.Client
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

// Kubectl will execute a kubectl command and panic if the script fails.
func (t *Context) Kubectl(args ...string) (stdout, stderr string) {
	actualArgs := append([]string{kubectlCmd}, args...)
	success, stdout, stderr := run(t.ctx, actualArgs)
	if glog.V(9) {
		glog.V(9).Infof("stdout: %v", stdout)
		glog.V(9).Infof("stderr: %v", stderr)
	}
	if !success {
		panic(errors.Errorf("Command %s failed, stdout: %s stderr: %s", strings.Join(args, " "), stdout, stderr))
	}
	return
}

// Apply runs kubectl apply -f on a given path.
func (t *Context) Apply(path string) {
	t.Kubectl("apply", "-f", path)
}

// DeleteSecret deletes a secret from Kubernetes.
func (t *Context) DeleteSecret(name, namespace string) error {
	t.Kubectl("delete", "secret", fmt.Sprintf("-n=%v", namespace), name)
	// TODO(filmil): Needs to pipe an error out.
	return nil
}

// DeleteConfigmap deletes a configmap from Kubernetes.
func (t *Context) DeleteConfigmap(name, namespace string) error {
	t.Kubectl("delete", "configmap", fmt.Sprintf("-n=%v", namespace), name)
	return nil
}

func (t *Context) CreateSecretGenericFromFile(name, namespace string, filenames ...string) error {
	args := []string{
		"create", "secret", "generic", fmt.Sprintf("-n=%v", namespace), name,
	}
	for _, fn := range filenames {
		args = append(args, fmt.Sprintf("--from-file=%q", filepath.Clean(fn)))
	}
	t.Kubectl(args...)
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
	t.Kubectl(args...)
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
	stdout, stderr := t.Kubectl("version", "-o", "json")
	if glog.V(8) {
		glog.Infof("stdout: %v\nstderr:%v", stdout, stderr)
	}
	if stderr != "" {
		glog.Warningf("GetClusterVersion(): nonempty stderr: %v", stderr)
	}
	var vs versionOutput
	json.Unmarshal([]byte(stdout), &vs)
	glog.Warningf("vs: %+v", vs)
	// GitVersion is of the form "v1.9.2-something"
	version := vs.ServerVersion.GitVersion[1:]
	v, err := semver.Parse(version)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "while getting version")
	}
	return v, nil
}

type ClusterList struct {
	// Clusters is the list of clusters available to the user, keyed by the
	// name of the context
	Clusters map[string]string

	// Current the name of the context marked as "current" in the Clusters.
	Current string
}

// SetContext sets the cluster context to the context with the given name.  The
// named context must exist.
func (t *Context) SetContext(name string) error {
	_, stderr := t.Kubectl("config", "use-context", name)
	if stderr != "" {
		return errors.Errorf("nonempty stderr: %v", stderr)
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

func run(ctx context.Context, args []string) (bool, string, string) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exec.NewRedirected(os.Stdin, stdout, stderr).Run(ctx, args...)
	return true, stdout.String(), stderr.String()
}

func (t *Context) waitForDeployment(deadline time.Time, namespace string, name string) {
	for time.Now().Before(deadline) {
		deployment, err := t.Kubernetes().ExtensionsV1beta1().Deployments(
			namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			panic(errors.Wrapf(err, "Error getting deployment %s", name))
		}
		glog.V(2).Infof(
			"Deployment %s replicas %d, available %d", name, deployment.Status.Replicas, deployment.Status.AvailableReplicas)
		if deployment.Status.AvailableReplicas == deployment.Status.Replicas {
			glog.V(1).Infof("Deployment %s available", name)
			return
		}
		time.Sleep(time.Millisecond * 250)
	}
	panic(errors.Errorf("Deployment %s failed to become available before deadline", name))
}

// WaitForDeployments waits for deployments to be available
func (t *Context) WaitForDeployments(timeout time.Duration, deployments ...string) {
	deadline := time.Now().Add(timeout)
	for _, deployment := range deployments {
		parts := strings.Split(deployment, ":")
		namespace := parts[0]
		name := parts[1]
		t.waitForDeployment(deadline, namespace, name)
	}
}

// Kubernetes returns the underlying Kubernetes client.
func (c *Context) Kubernetes() kubernetes.Interface {
	return c.client.Kubernetes()
}

// PolicyHierarchy returns the policyhierarchy client interface
func (t *Context) PolicyHierarchy() policyhierarchy.Interface {
	return t.client.PolicyHierarchy()
}
