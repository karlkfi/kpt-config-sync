/*
Copyright 2017 The Nomos Authors.
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
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/nomos/pkg/testing/e2e/testregistry"

	_ "github.com/google/nomos/pkg/testing/e2e/admissiontests" // Register admission tests
	_ "github.com/google/nomos/pkg/testing/e2e/syncertests"    // Register syncer tests
	"github.com/google/nomos/pkg/testing/e2e/testcontext"
)

// TestOptions contains options for executing the test cases.
type TestOptions struct {
	RepoDir             string
	TestFunctionPattern string
	LegacyTestFunctions string
	SkipSetup           bool
	SkipCleanup         bool
}

// RunTests will run all testcases. This should be invoked by main after flag parsing.
func RunTests(testOptions TestOptions) {
	ctx := context.Background()
	execContext, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		select {
		case <-c:
			cancelFunc()
			signal.Reset(os.Interrupt, syscall.SIGTERM)
		case <-execContext.Done():
		}
	}()

	testContext := testcontext.New(execContext, testOptions.RepoDir)

	if !testOptions.SkipSetup {
		legacyCleanup(testContext)
		legacySetup(testContext)
	}

	var runTests, runLegacyTests bool
	switch {
	case testOptions.TestFunctionPattern == "" && testOptions.LegacyTestFunctions == "":
		runTests = true
		runLegacyTests = true
	case testOptions.TestFunctionPattern == "" && testOptions.LegacyTestFunctions != "":
		runTests = false
		runLegacyTests = true
	case testOptions.TestFunctionPattern != "" && testOptions.LegacyTestFunctions == "":
		runTests = true
		runLegacyTests = false
	case testOptions.TestFunctionPattern != "" && testOptions.LegacyTestFunctions != "":
		runTests = true
		runLegacyTests = true
	}

	if runTests {
		fmt.Printf("RUNNING TESTCASES\n")
		allTests := testregistry.TestCases(testOptions.TestFunctionPattern)
		for _, testCase := range allTests {
			testCase.Test(testContext)
		}
		fmt.Printf("DONE RUNNING TESTCASES\n")
	}

	// TODO: Insert additional test cases here.
	if runLegacyTests {
		fmt.Printf("RUNNING LEGACY TESTCASES\n")
		os.Setenv("TEST_FUNCTIONS", testOptions.LegacyTestFunctions)
		testContext.RunBashOrDie(filepath.Join(testOptions.RepoDir, "e2e/e2e-legacy.sh"))
		fmt.Printf("DONE RUNNING LEGACY TESTCASES\n")
	}

	if !testOptions.SkipCleanup {
		legacyCleanup(testContext)
	}
}

func legacyCleanup(testContext *testcontext.TestContext) {
	testContext.RunBashOrDie(testContext.Repo("e2e/e2e-legacy-cleanup.sh"))
}

func legacySetup(testContext *testcontext.TestContext) {
	testContext.RunBashOrDie(testContext.Repo("e2e/e2e-legacy-setup.sh"))

	// TODO: make "deployment" a first class thing here.
	testContext.WaitForDeployments(
		time.Second*90, "nomos-system:syncer", "nomos-system:resourcequota-admission-controller")
}
