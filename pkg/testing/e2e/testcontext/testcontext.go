/*
Copyright 2017 The Stolos Authors.
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

package testcontext

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/policyhierarchy"
	"github.com/google/stolos/pkg/client/restconfig"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// TestContext is a context for e2e tests with some common functions attached to it.
type TestContext struct {
	execContext context.Context
	repoPath    string
	client      *meta.Client
}

// New returns a new test context
func New(execContext context.Context, testDir string) *TestContext {
	restConfig, err := restconfig.NewKubectlConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to get restconfig"))
	}

	client := meta.NewForConfigOrDie(restConfig)
	return &TestContext{
		execContext: execContext,
		client:      client,
	}
}

// RunBashOrDie will execute a bash script and panic if the script fails.
func (t *TestContext) RunBashOrDie(scriptPath string) {
	var err error
	cmd := exec.CommandContext(t.execContext, "/bin/bash", "-c", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(errors.Wrapf(err, "Script %s exited non-zero", scriptPath))
	}
}

// Kubectl will execute a kubectl command and panic if the script fails.
func (t *TestContext) Kubectl(args ...string) {
	actualArgs := append([]string{"kubectl"}, args...)
	success, stdout, stderr := t.run(actualArgs)
	if !success {
		panic(errors.Errorf("Command %s failed, stdout: %s stderr: %s", strings.Join(args, " "), stdout, stderr))
	}
}

// KubectlApply runs kubectl apply -f on a relative path in the repo.
func (t *TestContext) KubectlApply(path string) {
	t.Kubectl("apply", "-f", t.Repo(path))
}

// Repo returns the path to a relative path in the repo.
func (t *TestContext) Repo(relativePath string) string {
	return filepath.Join(t.repoPath, relativePath)
}

func (t *TestContext) run(args []string) (bool, string, string) {
	if len(args) == 0 {
		panic(errors.Errorf("Cannot run command with 0 parameters"))
	}

	glog.V(1).Infof("Running command: %s", strings.Join(args, " "))

	cmd := exec.CommandContext(t.execContext, args[0], args[1:]...)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stdout
	cmd.Stdout = stderr
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, stdout.String(), stderr.String()
		}
		panic(errors.Wrapf(err, "Failed to properly execute command %s", strings.Join(args, " ")))
	}
	return true, stdout.String(), stderr.String()
}

func (t *TestContext) waitForDeployment(deadline time.Time, namespace string, name string) {
	for time.Now().Before(deadline) {
		deployement, err := t.client.Kubernetes().ExtensionsV1beta1().Deployments(
			namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			panic(errors.Wrapf(err, "Error getting deployment %s", name))
		}
		glog.V(2).Infof(
			"Deployment %s replicas %d, availalbe %d", name, deployement.Status.Replicas, deployement.Status.AvailableReplicas)
		if deployement.Status.AvailableReplicas == deployement.Status.Replicas {
			glog.V(1).Infof("Deployment %s available", name)
			return
		}
		time.Sleep(time.Millisecond * 250)
	}
	panic(errors.Errorf("Deployment %s failed to become available before deadline", name))
}

// WaitForDeployments waits for deployments to be available
func (t *TestContext) WaitForDeployments(timeout time.Duration, deployments ...string) {
	deadline := time.Now().Add(timeout)
	for _, deployment := range deployments {
		parts := strings.Split(deployment, ":")
		namespace := parts[0]
		name := parts[1]
		t.waitForDeployment(deadline, namespace, name)
	}
}

// Kubernetes returns the kubernets client inteface
func (t *TestContext) Kubernetes() kubernetes.Interface {
	return t.client.Kubernetes()
}

// PolicyHierarchy returns the policyhierarchy client interface
func (t *TestContext) PolicyHierarchy() policyhierarchy.Interface {
	return t.client.PolicyHierarchy()
}

// WaitForExists will wait until the returned error is nil while ignoring IsNotFound errors.
func (t *TestContext) WaitForExists(timeout time.Duration, functions ...func() error) {
	predicate := func(err error) bool {
		if err == nil {
			return true
		}
		if api_errors.IsNotFound(err) {
			return false
		}
		panic(errors.Wrapf(err, "WaitForExists encountered error other than not found"))
	}
	t.waitForCondition(timeout, predicate, functions)
}

// WaitForNotFound will wait until the resource returns IsNotFound error.
func (t *TestContext) WaitForNotFound(timeout time.Duration, functions ...func() error) {
	predicate := func(err error) bool {
		if err == nil {
			return false
		}
		if api_errors.IsNotFound(err) {
			return true
		}
		panic(errors.Wrapf(err, "WaitForNotFound encountered error other than not found"))
	}

	t.waitForCondition(timeout, predicate, functions)
}

// waitForCondition will wait until all functions have satisfied the predicate or panic.
func (t *TestContext) waitForCondition(
	timeout time.Duration, predicate func(error) bool, functions []func() error) {
	deadline := time.Now().Add(timeout)
	for _, function := range functions {
		for time.Now().Before(deadline) {
			if predicate(function()) {
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
	panic(errors.Errorf("Predicate did not return true before deadline."))
}
