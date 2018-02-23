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

// Controller responsible for importing policies from a Git repo and materializing CRDs
// on the local cluster.
package main

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/policyimporter/filesystem"
	"github.com/google/stolos/pkg/util/log"
)

var inCluster = flag.Bool("in-cluster", true,
	"Whether running in a Kubernetes clsuter")
var gitDir = flag.String("git-dir", "/repo/rev",
	"Absolute path to the git repo")
var policyDirRelative = flag.String("policy-dir", envString("POLICY_DIR", ""),
	"Relative path of root policy directory in the repo")
var pollPeriod = flag.Duration("poll-period", time.Second*5,
	"Poll period for checking if --git-dir target directly has changed")

func envString(key, def string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return def
}

func main() {
	flag.Parse()
	log.Setup()

	glog.Infof("Starting GitPolicyImporter...")

	policyDir := path.Join(*gitDir, *policyDirRelative)
	glog.Infof("Policy dir: %s", policyDir)

	parser, err := filesystem.NewParser(*inCluster)
	if err != nil {
		glog.Fatalf("Failed to create parser: %v", err)
	}

	ticker := time.NewTicker(*pollPeriod)
	currentDir := ""
	for range ticker.C {
		newDir, err := filepath.EvalSymlinks(policyDir)
		if err != nil {
			glog.Fatal(err)
		}

		if currentDir == newDir {
			// No new commits, nothing to do.
			continue
		}

		glog.Infof("Resolved policy dir: %s", newDir)
		currentDir = newDir

		policies, err := parser.Parse(newDir)
		if err != nil {
			glog.Warningf("Failed to parse: %v", err)
			continue
		}

		glog.Infof("%#v", policies)
	}
}
