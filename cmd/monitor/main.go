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

// Controller responsible for monitoring the state of Nomos resources on the cluster.
package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/monitor"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func main() {
	flag.Parse()
	log.Setup()

	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("Failed to create rest config: %v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Fatalf("Failed to create manager: %v", err)
	}

	go service.ServeMetrics()

	// Setup all Controllers
	if err := monitor.AddToManager(mgr); err != nil {
		glog.Fatalf("Failed to add controller to manager: %v", err)
	}

	// Start the Manager.
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatalf("Error starting controller: %v", err)
	}
}
