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

	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/monitor"
	"github.com/google/nomos/pkg/monitor/args"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()
	log.Setup()

	config, err := restconfig.NewRestConfig()
	if err != nil {
		panic(errors.Wrapf(err, "failed to create rest config"))
	}

	go service.ServeMetrics()

	injectArgs := args.CreateInjectArgs(config)
	runArgs := run.RunArguments{Stop: signals.SetupSignalHandler()}
	controller := monitor.NewController(injectArgs)
	controller.Start(runArgs)
	<-runArgs.Stop
}
