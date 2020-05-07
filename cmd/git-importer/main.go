// Controller responsible for importing policies from a Git repo and materializing CRDs
// on the local cluster.
package main

import (
	"flag"

	"github.com/google/nomos/pkg/configsync"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

func main() {
	flag.Parse()
	log.Setup()
	go service.ServeMetrics()

	configsync.RunImporter()
}
