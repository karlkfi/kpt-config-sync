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

// Controller responsible for fetching PolicyNode(s) from a remote cluster API server to local cluster.
package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/google/stolos/pkg/policyimporter/remotecluster"
	"github.com/google/stolos/pkg/service"
	"github.com/google/stolos/pkg/util/log"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()
	log.Setup()

	glog.Infof("Starting RemoteClusterPolicyImporter...")

	config, err := restconfig.NewRestConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create rest config"))
	}

	remoteConfig, err := restconfig.NewRemoteClusterConfig()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create remote rest config"))
	}

	client, err := meta.NewForConfig(config)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create client"))
	}

	remoteClient, err := meta.NewForConfig(remoteConfig)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create remote client"))
	}

	stopChannel := make(chan struct{})

	pnc := remotecluster.NewController(client, remoteClient, stopChannel)
	pnc.Run()
	service.WaitForShutdownWithChannel(stopChannel)
	pnc.Stop()
}
