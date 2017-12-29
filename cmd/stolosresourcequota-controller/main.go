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

	"github.com/google/stolos/pkg/client/meta"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/google/stolos/pkg/resource-quota"
	"github.com/google/stolos/pkg/service"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	glog.Infof("Starting Stolos Resource Quota Controller...")

	config, err := restconfig.NewRestConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create rest config"))
	}

	client, err := meta.NewForConfig(config)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create client"))
	}

	stopChannel := make(chan struct{})

	quotaController := resource_quota.NewController(client, stopChannel)
	quotaController.Run()
	service.WaitForShutdownWithChannel(stopChannel)

	quotaController.Stop()
}
