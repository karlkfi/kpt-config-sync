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

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// TestContext contains options for executing the test cases.
type TestContext struct {
	TestDir             string
	TestFunctionPattern string
	LegacyTestFunctions string
	SkipSetup           bool
	SkipCleanup         bool
}

// RunTests will run all testcases.
func RunTests(testContext TestContext) {
	ctx := context.Background()
	execContext, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	if !testContext.SkipSetup {
		runBash(execContext, filepath.Join(testContext.TestDir, "e2e-legacy-setup.sh"))
	}

	// TODO: Insert additional test cases here.
	runBash(execContext, filepath.Join(testContext.TestDir, "e2e-legacy.sh"))

	if !testContext.SkipCleanup {
		runBash(execContext, filepath.Join(testContext.TestDir, "e2e-legacy-cleanup.sh"))
	}
}

func runBash(execContext context.Context, scriptPath string) {
	var err error
	cmd := exec.CommandContext(execContext, "/bin/bash", "-c", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		panic(errors.Wrapf(err, "%s failed to start", scriptPath))
	}

	err = cmd.Wait()
	if err != nil {
		panic(errors.Wrapf(err, "%s failed to finish", scriptPath))
	}
}
