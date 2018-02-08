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

// Controller responsible for importing policies from a Git repo and materializing PolicyNodes
// on the local cluster.
package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/util/log"
)

const (
	// Poll period for checking if symlink was updated by git-sync.
	// Worst case it takes (pollPeriod + GIT_SYNC_WAIT) to detect changes.
	pollPeriod = time.Second * 5
	// Symlink to the git repo created by git-sync.
	gitDir     = "/repo/rev"
)

var policyDirRelative = flag.String("policy-dir", envString("POLICY_DIR", ""),
	"Relative path of root policy directory in the repo")

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

	policyDir := path.Join(gitDir, *policyDirRelative)
	glog.Infof("Root policyspace dir: %s", policyDir)

	ticker := time.NewTicker(pollPeriod)
	currentDir := ""
	for range ticker.C {
		newDir, err := filepath.EvalSymlinks(gitDir)
		if err != nil {
			glog.Fatal(err)
		}

		if currentDir == newDir {
			// No new commits, nothing to do.
			continue
		}

		glog.Infof("New rev dir: %s", newDir)
		currentDir = newDir

		files, err := ioutil.ReadDir(policyDir)
		if err != nil {
			glog.Fatal(err)
		}

		for _, file := range files {
			glog.Info(file.Name())
		}

		// TODO(frankf): Do useful work.
	}
}
