/*
Copyright 2017 The Stolos Authors.
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
	"fmt"
	"net/http"

	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/restconfig"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/service"
	"github.com/google/stolos/pkg/syncer"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsPort = flag.Int("metrics-port", 8675, "The port to export prometheus metrics on.")
)

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	config, err := restconfig.NewRestConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create rest config"))
	}

	client, err := meta.NewForConfig(config)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create client"))
	}

	// Expose prometheus metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), nil)
		if err != nil {
			glog.Fatalf("HTTP ListenAndServe: %+v", err)
		}
	}()

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
