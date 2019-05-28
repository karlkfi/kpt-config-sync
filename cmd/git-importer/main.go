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
		&genericclioptions.ConfigFlags{}, filesystem.ParserOpt{Validate: true, Extension: &filesystem.NomosVisitorProvider{}})
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
