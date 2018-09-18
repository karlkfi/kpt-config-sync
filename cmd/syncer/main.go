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

// Command line util to sync the PolicyNode custom resource to the active namespaces.
package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/syncercontroller"
	"github.com/google/nomos/pkg/util/log"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
)

func main() {
	flag.Parse()
	log.Setup()

	restConfig, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %v", err)
	}

	go service.ServeMetrics()

	injectArgs := args.CreateInjectArgs(restConfig)
	syncerController := syncercontroller.New(injectArgs)

	runArgs := run.RunArguments{Stop: signals.SetupSignalHandler()}
	err = syncerController.Start(runArgs)
	if err != nil {
		glog.Fatalf("failed to start controller: %v", err)
	}

	<-runArgs.Stop
}
