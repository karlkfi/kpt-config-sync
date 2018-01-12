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

// The main driver of the Stolos resource quota controller for policyspace quota.
// See more details in stolos_quota_controller.
package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/google/stolos/pkg/resource-quota"
	"github.com/google/stolos/pkg/service"
	"github.com/google/stolos/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsPort = flag.Int("metrics-port", 8675, "The port to export prometheus metrics on.")
)

func main() {
	flag.Parse()
	log.Setup()

	glog.Infof("Starting Stolos Resource Quota Controller...")

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
			glog.Fatalf("HTTP ListenAndServe for metrics: %+v", err)
		}
	}()

	stopChannel := make(chan struct{})

	quotaController := resource_quota.NewController(client, stopChannel)
	quotaController.Run()
	service.WaitForShutdownWithChannel(stopChannel)

	quotaController.Stop()
}
