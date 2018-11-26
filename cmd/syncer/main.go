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

// Controllers responsible for syncing declared resources to the cluster.
package main

import (
	"flag"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/generic-syncer/metasync"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var (
	resyncPeriod = flag.Duration(
		"resync_period", time.Minute, "The resync period for the syncer system")
)

func main() {
	flag.Parse()
	log.Setup()

	go service.ServeMetrics()

	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %+v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{SyncPeriod: resyncPeriod})
	if err != nil {
		glog.Fatalf("Failed to create manager: %+v", err)
	}

	mgrStopChannel := signals.SetupSignalHandler()
	// Set up Scheme for generic resources.
	if err := metasync.AddMetaController(mgr, mgrStopChannel); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	// Start the Manager.
	if err := mgr.Start(mgrStopChannel); err != nil {
		glog.Fatalf("Error starting controller: %+v", err)
	}
}
