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
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/cli"

	_ "github.com/google/stolos/pkg/cli/registration"
)

func main() {
	flag.Parse()
	context, err := cli.NewCommandContext()
	if err != nil {
		glog.Fatalf("Failed to create command context: %s", err)
	}

	args := flag.Args()
	if err := context.Invoke(args); err != nil {
		glog.Errorf("Failed to invoke command %q: %s", strings.Join(args, " "), err)
		os.Exit(1)
	}
}
