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

// This is the syncer service, it will take config spec from the custom resource then apply it
// to namespaces while watching for updates.
package main

import (
	"flag"

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/service"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()

	// TODO: add flag for using service account credentials, flag for server address.
	clusterClient, err := client.NewMiniKubeClient()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create kube client"))
	}

	// TODO: add --daemon flag as a control knob for run once vs continuously
	service.WaitForShutdownSignal(clusterClient.RunSyncerDaemon())
}
