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
// Reviewed by sunilarora

// Package options serves to provide a common location for flag definitions during the syncer
// rewrite transition.
package options

import (
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
)

var (
	resyncPeriod = flag.Duration(
		"resync_period", time.Minute, "The resync period for the syncer system")

	dryRun = flag.Bool(
		"dry_run", false, "Don't perform actions, just log what would have happened")

	workerNumRetries = flag.Int(
		"worker_num_retries", 3, "Number of retries for an action before giving up.")

	gcpMode = flag.Bool(
		"gcpMode", false, "Runs syncer in GCP mode.")
)

// Options are the flag value options for the syncer
type Options struct {
	ResyncPeriod     time.Duration // flag resync_period
	DryRun           bool          // flag dry_run
	WorkerNumRetries int           // flag worker_num_retries
	GCPMode          bool          // flag to indicate the sync is from GCP.
}

// FromFlagsAndEnv returns a copy of the options from flag and environment values.
func FromFlagsAndEnv() Options {
	var sycnerGCPMode bool
	envGcpMode := os.Getenv("GCP_MODE")
	if envGcpMode == "true" || *gcpMode {
		sycnerGCPMode = true
		glog.Info("Running in GCP mode.")
	}

	return Options{
		ResyncPeriod:     *resyncPeriod,
		DryRun:           *dryRun,
		WorkerNumRetries: *workerNumRetries,
		GCPMode:          sycnerGCPMode,
	}
}
