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
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/util/log"
)

const (
	pollPeriod = time.Second * 30
	gitDir     = "/repo/rev"
)

func main() {
	flag.Parse()
	log.Setup()

	glog.Infof("Starting GitPolicyImporter...")

	// TODO(frankf): Detect symlink target change to figure out if there's new commits.
	ticker := time.NewTicker(pollPeriod)
	for range ticker.C {
		files, err := ioutil.ReadDir(gitDir)
		if err != nil {
			glog.Fatal(err)
		}

		for _, file := range files {
			glog.Info(file.Name())
		}

		// TODO(frankf): Do useful work.
	}

}
