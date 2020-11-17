// Controller responsible for importing policies from a Git repo and materializing CRDs
// on the local cluster.
package main

import (
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/configsync"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

var (
	watchDirectory = flag.String("watch-directory", "", "Watch a directory and log filesystem changes instead of running as importer")
	watchPeriod    = flag.Duration("watch-period", getEnvDuration("WATCH_PERIOD", time.Second), "Period at which to poll the watch directory for changes.")
)

func getEnvDuration(key string, defaultDuration time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultDuration
	}

	duration, err := time.ParseDuration(val)
	if err != nil {
		glog.Errorf("Failed to parse duration %q from env var %s: %s", val, key, err)
		return defaultDuration
	}
	return duration
}

func main() {
	flag.Parse()
	log.Setup()
	if *watchDirectory != "" {
		configsync.DirWatcher(*watchDirectory, *watchPeriod)
		os.Exit(0)
	}

	go service.ServeMetrics()
	configsync.RunImporter()
}
