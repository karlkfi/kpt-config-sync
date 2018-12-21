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

// Controller responsible for importing policies from a Git repo and materializing CRDs
// on the local cluster.
package main

import (
	"flag"
	"path"
	"time"

	"os"

	"github.com/golang/glog"
	"github.com/google/nomos/cmd/nomos/parse"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policyimporter/filesystem"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
)

const bespinEnv = "NOMOS_ENABLE_BESPIN"

var (
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directly has changed")
	// Check for a set environment variable instead of using a flag so as not to expose
	// this WIP externally.
	_, bespin = os.LookupEnv(bespinEnv)
)

func main() {
	flag.Parse()
	log.Setup()

	config, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %+v", err)
	}

	client, err := meta.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %+v", err)
	}

	policyDir := path.Join(*gitDir, *policyDirRelative)
	glog.Infof("Policy dir: %s", policyDir)

	e := &parse.Ext{}
	if bespin {
		e = &parse.Ext{VP: filesystem.BespinVisitors, Syncs: filesystem.BespinSyncs}
	}
	parser, err := filesystem.NewParser(&genericclioptions.ConfigFlags{}, filesystem.ParserOpt{Validate: true, Extension: e})
	if err != nil {
		glog.Fatalf("Failed to create parser: %+v", err)
	}

	go service.ServeMetrics()

	stopChan := make(chan struct{})

	c := filesystem.NewController(policyDir, *pollPeriod, parser, client, stopChan)
	go service.WaitForShutdownSignalCb(stopChan)
	if err := c.Run(); err != nil {
		glog.Fatalf("Failure running controller: %+v", err)
	}
	glog.Info("Exiting")
}
