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

// Command line utility to check whether a directory contains a valid policy hierarchy.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/pkg/errors"
)

func printErrAndDie(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

const usage = `Stolosvet is a tool for validating a policy directory tree.

Usage:

	stolosvet DIRECTORY

DIRECTORY is the root policy directory. This is typically a subdirectory in a Git repo.

Example:

	stolosvet my-repo/policy-dir
	stolosvet /path/to/my-repo/policy-dir

Options:

`

func main() {
	flag.Usage = func() {
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

	p, err := filesystem.NewParser(false)
	if err != nil {
		printErrAndDie(errors.Wrap(err, "Failed to create parser"))
	}

	if _, err := p.Parse(dir); err != nil {
		printErrAndDie(errors.Wrap(err, "Found issues"))
	}
}
