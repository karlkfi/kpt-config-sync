// Controllers responsible for syncing declared resources to the cluster.
package main

import (
	"flag"

	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

func main() {
	flag.Parse()
	log.Setup()

	// In this transitional state, we want main to never exit so the Pod continues
	// to list itself as running. This keeps us from having to modify any of our
	// YAML while testing this state.
	service.ServeMetrics()
}
