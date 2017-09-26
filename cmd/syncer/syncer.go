/*
Copyright 2017 The Kubernetes Authors.
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

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/meta"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/restconfig"

	"github.com/golang/glog"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/service"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/syncer"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()

	config, err := restconfig.NewRestConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create rest config"))
	}

	client, err := meta.NewForConfig(config)
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
