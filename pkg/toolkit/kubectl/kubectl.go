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
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/policyhierarchy"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/google/stolos/pkg/toolkit/exec"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	kubectlCmd = exec.RequireProgram("kubectl")
)

// Context contains the runtime context for interacting with the Kubernetes client
// and server.
type Context struct {
	ctx    context.Context
	client *meta.Client
}

// New creates a new kubernetes context.
func New(ctx context.Context) *Context {
	restConfig, err := restconfig.NewKubectlConfig()
	if err != nil {
		panic(errors.Wrapf(err, "failed to get restconfig"))
	}
	return &Context{ctx, meta.NewForConfigOrDie(restConfig)}
}

// Kubectl will execute a kubectl command and panic if the script fails.
func (t *Context) Kubectl(args ...string) {
	actualArgs := append([]string{kubectlCmd}, args...)
	success, stdout, stderr := run(t.ctx, actualArgs)
	if !success {
		panic(errors.Errorf("Command %s failed, stdout: %s stderr: %s", strings.Join(args, " "), stdout, stderr))
	}
}

// Apply runs kubectl apply -f on a given path.
func (t *Context) Apply(path string) {
	t.Kubectl("apply", "-f", path)
}

func run(ctx context.Context, args []string) (bool, string, string) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exec.NewRedirected(os.Stdin, stdout, stderr).Run(ctx, args...)
	return true, stdout.String(), stderr.String()
}

func (t *Context) waitForDeployment(deadline time.Time, namespace string, name string) {
	for time.Now().Before(deadline) {
		deployement, err := t.client.Kubernetes().ExtensionsV1beta1().Deployments(
			namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			panic(errors.Wrapf(err, "Error getting deployment %s", name))
		}
		glog.V(2).Infof(
			"Deployment %s replicas %d, available %d", name, deployement.Status.Replicas, deployement.Status.AvailableReplicas)
		if deployement.Status.AvailableReplicas == deployement.Status.Replicas {
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
