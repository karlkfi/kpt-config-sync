package main

import (
	"flag"
	"os"

	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

var (
	reconcilerScope = flag.String("reconciler-scope", os.Getenv("RECONCILER_SCOPE"), "Scope of the reconciler (either a namespace or ':root'")

	fightDetectionThreshold = flag.Float64(
		"fight_detection_threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")
)

func main() {
	flag.Parse()
	log.Setup()

	go service.ServeMetrics()

	opts := reconciler.Options{
		ReconcilerScope:         *reconcilerScope,
		FightDetectionThreshold: *fightDetectionThreshold,
	}
	reconciler.Run(opts)
}
