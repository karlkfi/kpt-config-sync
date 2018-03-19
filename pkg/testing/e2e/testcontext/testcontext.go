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
	"path/filepath"
	"time"

	"github.com/google/stolos/pkg/client/policyhierarchy"
	"github.com/google/stolos/pkg/toolkit/bash"
	"github.com/google/stolos/pkg/toolkit/exec"
	"github.com/google/stolos/pkg/toolkit/kubectl"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"
)

// TestContext is a context for e2e tests with some common functions attached to it.
type TestContext struct {
	execContext context.Context
	repoPath    string
	kubeClient  *kubectl.Context
}

// New returns a new test context
func New(execContext context.Context, testDir string) *TestContext {
	return &TestContext{
		execContext: execContext,
		kubeClient:  kubectl.New(execContext),
	}
}

// RunBashOrDie will execute a bash script and panic if the script fails.
func (t *TestContext) RunBashOrDie(scriptPath string) {
	bash.RunOrDie(t.execContext, scriptPath)
}

// Repo returns the path to a relative path in the repo.
func (t *TestContext) Repo(relativePath string) string {
	return filepath.Join(t.repoPath, relativePath)
}

// KubectlApply runs kubectl apply -f on a relative path in the repo.
func (t *TestContext) KubectlApply(path string) {
	t.kubeClient.Apply(t.Repo(path))
}

func (t *TestContext) run(args []string) (bool, string, string) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exec.NewRedirected(os.Stdin, stdout, stderr).Run(t.execContext, args...)
	return true, stdout.String(), stderr.String()
}

// WaitForDeployments waits for deployments to be available
func (t *TestContext) WaitForDeployments(timeout time.Duration, deployments ...string) {
	if err := t.kubeClient.WaitForDeployments(timeout, deployments...); err != nil {
		// It is OK to panic in an end-to-end test harness.
		panic(errors.Wrapf(err, "while waiting for: %v", deployments))
	}
}

// Kubernetes returns the kubernets client inteface
func (t *TestContext) Kubernetes() kubernetes.Interface {
	return t.kubeClient.Kubernetes()
}

// Predicate is used to wait for conditions, and can be named to ease diagnostics.
type Predicate interface {
	Name() string
	// Eval checks the error returned by the API call and returns true or false.
	// In case of an unexpected error, Eval should wrap the error using errors.Wrap
	// and panic.
	Eval(error) bool
}

type predicateFunction struct {
	name string
	f    func(error) bool
}

func (p predicateFunction) Name() string        { return p.name }
func (p predicateFunction) Eval(err error) bool { return p.f(err) }

func NewPredicate(name string, f func(error) bool) *predicateFunction {
	return &predicateFunction{name, f}
}

// PolicyHierarchy returns the policyhierarchy client interface
func (t *TestContext) PolicyHierarchy() policyhierarchy.Interface {
	return t.kubeClient.PolicyHierarchy()
}

// WaitForExists will wait until the returned error is nil while ignoring IsNotFound errors.
func (t *TestContext) WaitForExists(timeout time.Duration, functions ...func() error) {
	predicate := NewPredicate("WaitForExists", func(err error) bool {
		if err == nil {
			return true
		}
		if api_errors.IsNotFound(err) {
			return false
		}
		panic(errors.Wrapf(err, "WaitForExists encountered error other than not found"))
	})
	t.waitForCondition(timeout, predicate, functions)
}

// WaitForNotFound will wait until the resource returns IsNotFound error.
func (t *TestContext) WaitForNotFound(timeout time.Duration, functions ...func() error) {
	predicate := NewPredicate("WaitForNotFound", func(err error) bool {
		if err == nil {
			return false
		}
		if api_errors.IsNotFound(err) {
			return true
		}
		panic(errors.Wrapf(err, "WaitForNotFound encountered error other than not found"))
	})

	t.waitForCondition(timeout, predicate, functions)
}

// waitForCondition will wait until all functions have satisfied the predicate or panic.
func (t *TestContext) waitForCondition(
	timeout time.Duration, predicate Predicate, functions []func() error) {
	deadline := time.Now().Add(timeout)
	for _, function := range functions {
		for time.Now().Before(deadline) {
			if predicate.Eval(function()) {
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
	panic(errors.Errorf("Predicate %q did not return true before deadline.", predicate.Name()))
}
