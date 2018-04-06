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

// Command line util to sync the PolicyNode custom resource to the active namespaces.
package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/syncer"
	"github.com/google/nomos/pkg/syncer/args"
	"github.com/google/nomos/pkg/syncer/syncercontroller"
	"github.com/google/nomos/pkg/util/log"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

var (
	useNewSyncer = flag.Bool("useNewSyncer", false, "use the new syncer")
)

func main() {
	flag.Parse()
	log.Setup()

	restConfig, err := restconfig.NewRestConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create rest config"))
	}

	go service.ServeMetrics()

	if *useNewSyncer {
		newSyncerMain(restConfig)
	} else {
		syncerMain(restConfig)
	}
}

func syncerMain(restConfig *rest.Config) {
	client, err := meta.NewForConfig(restConfig)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create client"))
	}

	stopChannel := make(chan struct{})
	errorCallback := func(err error) {
		glog.Errorf("Got error from error callback: %s", err)
		close(stopChannel)
	}

	clusterSyncer := syncer.New(client)
	clusterSyncer.Run(errorCallback)

	service.WaitForShutdownWithChannel(stopChannel)

	clusterSyncer.Stop()
	clusterSyncer.Wait()
}

func newSyncerMain(restConfig *rest.Config) {
	injectArgs := args.CreateInjectArgs(restConfig)
	syncerController := syncercontroller.New(injectArgs)

	runArgs := run.RunArguments{Stop: signals.SetupSignalHandler()}
	syncerController.Start(runArgs)

	<-runArgs.Stop
}
