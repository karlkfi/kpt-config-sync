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
	"context"
	"flag"
	"os"
	"path"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directory has changed")
	resyncPeriod      = flag.Duration("resync-period", time.Minute, "The resync period for the importer system")
	// TODO(129774660): Clean up after launching CRD syncing.
	enableCRDs = flag.Bool("enable_crds", true, "When true, enable syncing CRDs")
)

func main() {
	flag.Parse()
	log.Setup()

	config, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %+v", err)
	}

	client, err := meta.NewForConfig(config, resyncPeriod)
	if err != nil {
		glog.Fatalf("Failed to create client: %+v", err)
	}

	policyDir := path.Join(*gitDir, *policyDirRelative)
	glog.Infof("Policy dir: %s", policyDir)

	parser := filesystem.NewParser(
		&genericclioptions.ConfigFlags{}, filesystem.ParserOpt{Validate: true, Extension: &filesystem.NomosVisitorProvider{}, EnableCRDs: *enableCRDs})
	if err := parser.ValidateInstallation(); err != nil {
		glog.Fatalf("Found issues: %v", err)
	}

	go service.ServeMetrics()

	stopChan := make(chan struct{})

	c, err := filesystem.NewController(policyDir, *pollPeriod, parser, client, stopChan)
	if err != nil {
		glog.Fatalf("Failure creating controller: %v", err)
	}
	go service.WaitForShutdownSignalCb(stopChan)
	if err := c.Run(context.Background()); err != nil {
		glog.Fatalf("Failure running controller: %+v", err)
	}
	glog.Info("Exiting")
}
