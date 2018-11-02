// Reviewed by sunilarora
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

// Command line utility to check whether a directory contains a valid policy hierarchy.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"encoding/json"

	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

const usage = `Nomosvet is a tool for validating a policy directory tree.

Usage:

	nomosvet DIRECTORY

DIRECTORY is the root policy directory. This is typically a subdirectory in a Git repo.

Example:

	nomosvet my-repo/policy-dir
	nomosvet /path/to/my-repo/policy-dir

Options:

`

var validate = flag.Bool("validate", true, "If true, use a schema to validate the input")
var print = flag.Bool("print", false, "If true, print generated Nomos CRDs")

func printErrAndDie(err error) {
	// nolint: errcheck
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	flag.Usage = func() {
		// nolint: errcheck
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	dir, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to get absolute path"))
	}

	clientConfig, err := restconfig.NewClientConfig()
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to get kubectl config"))
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to get rest.Config"))
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to create client"))
	}

	p, err := filesystem.NewParser(
		clientConfig, client.Discovery(), filesystem.Validate(*validate))
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to create parser"))
	}

	policies, err := p.Parse(dir)
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Found issues"))
	}
	if *print {
		err := prettyPrint(policies)
		if err != nil {
			printErrAndDie(errors.Wrap(err, "Failed to print generated CRDs"))
		}
	}
}

func prettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return err
}
