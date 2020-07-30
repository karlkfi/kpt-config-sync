package main

import (
	"flag"
	"os"

	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

var (
	fightDetectionThreshold = flag.Float64(
		"fight_detection_threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")

	numWorkers = flag.Int("num_workers", 1, "Number of concurrent remediator workers to run at once.")

	reconcilerScope = flag.String("reconciler-scope", os.Getenv("RECONCILER_SCOPE"), "Scope of the reconciler (either a namespace or ':root').")
)

func main() {
	flag.Parse()
	log.Setup()

	go service.ServeMetrics()

	opts := reconciler.Options{
		FightDetectionThreshold: *fightDetectionThreshold,
		NumWorkers:              *numWorkers,
		ReconcilerScope:         *reconcilerScope,
	}
	reconciler.Run(opts)
}
