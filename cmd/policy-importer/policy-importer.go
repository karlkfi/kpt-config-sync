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

package main

import (
	"flag"

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/importer"
	"github.com/pkg/errors"
)

var flagSyncDir = flag.String(
	"sync_dir", "", "Directory to inspect for .yaml or .json file for populating the custom resource")
var flagDaemon = flag.Bool(
	"daemon", false, "Continuously watch sync_dir for changes, note that this will not actually daemonize the process")

func main() {
	flag.Parse()

	// TODO: add flag for service account credentials, flag for server address.
	kubeClient, err := client.NewMiniKubeClient()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to create kube client"))
	}

	importerInstance := importer.New(importer.ImporterOptions{
		Daemon:        *flagDaemon,
		ConfigDirPath: *flagSyncDir,
		Client:        kubeClient,
	})

	err = importerInstance.Run()
	if err != nil {
		panic(err)
	}
}
