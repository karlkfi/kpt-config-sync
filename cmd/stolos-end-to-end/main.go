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

package main

import (
	"flag"

	"github.com/google/stolos/pkg/testing/e2e"
	"github.com/pkg/errors"
)

var shellTestDir = flag.String(
	"test_dir", "", "Path to shell script test directory with legacy e2e tests")
var skipSetup = flag.Bool("skip_setup", false, "Skip test setup")
var skipCleanup = flag.Bool("skip_cleanup", false, "Skip test cleanup")
var legacyTestFunctions = flag.String(
	"legacy_test_functions", "", "Test functions to invoke, unset will invoke all")
var testFunctionPattern = flag.String(
	"test_function_pattern", "", "Test name pattern for functions to invoke")

func main() {
	flag.Parse()
	if *shellTestDir == "" {
		panic(errors.Errorf("-test_dir flag not set"))
	}

	e2e.RunTests(e2e.TestContext{
		TestDir:             *shellTestDir,
		SkipSetup:           *skipSetup,
		SkipCleanup:         *skipCleanup,
		LegacyTestFunctions: *legacyTestFunctions,
		TestFunctionPattern: *testFunctionPattern,
	})
}
