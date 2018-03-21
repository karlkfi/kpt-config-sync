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

// Package testregistry handles registration and filtering for test cases. Test cases must be
// registered using the "Register" function in order to be run.
package testregistry

import (
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/nomos/pkg/testing/e2e/testcontext"
)

// SetupFunc is the signature for a test setup function
type SetupFunc func(*testcontext.TestContext)

// TestCase is the signature for a test case
type TestCase func(*testcontext.TestContext)

// CleanupFunc is the signature for a test cleanup function
type CleanupFunc func(*testcontext.TestContext)

// RegisteredTestCase represents a single test case that will be run.
type RegisteredTestCase struct {
	name        string
	testFunc    TestCase
	setupFunc   SetupFunc
	cleanupFunc CleanupFunc
}

// Name is the full name of the testcase including path, eg github.com/...
func (s *RegisteredTestCase) Name() string {
	return s.name
}

// ShortName is the package name and function name of the test, eg, mytests.testSituationFoo
func (s *RegisteredTestCase) ShortName() string {
	idx := strings.LastIndex(s.name, "/")
	if idx < 0 {
		return s.name
	}
	return s.name[idx+1:]
}

// Test will invoke the test case with the given context
func (s *RegisteredTestCase) Test(t *testcontext.TestContext) {
	if s.setupFunc != nil {
		s.setupFunc(t)
	}
	s.testFunc(t)
	if s.cleanupFunc != nil {
		s.cleanupFunc(t)
	}
}

// allTests contains all registered test cases
var allTests []RegisteredTestCase

// TestCases returns testcases that will be run given the filter pattern.
func TestCases(filterPattern string) []RegisteredTestCase {
	if filterPattern == "" {
		return allTests
	}

	matchedTests := []RegisteredTestCase{}
	matcher := regexp.MustCompile(filterPattern)
	for _, testCase := range allTests {
		if matcher.MatchString(testCase.ShortName()) {
			matchedTests = append(matchedTests, testCase)
		}
	}
	return matchedTests
}

// Register regsiters test cases with the registry.
func Register(setup SetupFunc, cleanup CleanupFunc, testCases ...TestCase) {
	for _, testCase := range testCases {
		allTests = append(allTests, RegisteredTestCase{
			name:        runtime.FuncForPC(reflect.ValueOf(testCase).Pointer()).Name(),
			testFunc:    testCase,
			setupFunc:   setup,
			cleanupFunc: cleanup,
		})
	}
}
