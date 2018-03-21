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
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

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

	config, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %v", err)
	}

	client, err := meta.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	policyDir := path.Join(*gitDir, *policyDirRelative)
	glog.Infof("Policy dir: %s", policyDir)

	parser, err := filesystem.NewParser(true)
	if err != nil {
		glog.Fatalf("Failed to create parser: %v", err)
	}

	stopChan := make(chan struct{})
	c := filesystem.NewController(policyDir, *pollPeriod, parser, client, stopChan)
	go service.WaitForShutdownSignalCb(stopChan)
	if err := c.Run(); err != nil {
		glog.Fatalf("Failure running controller: %v", err)
	}
	glog.Info("Exiting")
}
