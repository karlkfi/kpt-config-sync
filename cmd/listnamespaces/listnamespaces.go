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

// An example for usage of the client package, here mainly for manual testing during
// bootstrapping.  Expect this to go away at some point.
package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/pkg/errors"
)

var (
	flagSyncNamespacesDemo = flag.Bool(
		"sync_namespaces_demo", false, "demonstrate syncing namespaces")

	flagClientMode = flag.String(
		"client_mode", "minikube", "How to connect to the service, options are"+
			" minikube, serviceaccount (sa for short)")

	flagSecretPath = flag.String("secret_path", "", "Path to the secret yaml file")
)

func main() {
	flag.Parse()

	var clusterClient *client.Client
	var err error
	switch *flagClientMode {
	case "sa":
		fallthrough
	case "serviceaccount":
		clusterClient, err = client.NewServiceAccountClient(*flagSecretPath)

	case "minikube":
		clusterClient, err = client.NewClient(*flagClientMode)

	default:
		panic(errors.Errorf("No client %s", *flagClientMode))
	}

	if err != nil {
		glog.Fatalf("Error creating client: %v\n", err)
	}
	state, err := clusterClient.GetState()
	if err != nil {
		glog.Fatalf("Error fetching state: %v\n", err)
	}

	fmt.Printf("Found namespaces:\n")
	for _, namespace := range state.Namespaces {
		fmt.Printf(" %s\n", namespace)
	}

	if *flagSyncNamespacesDemo {
		syncNamespacesDemo(clusterClient)
	}
}

// Demonstrate namespace sync by syncing list of NS (create), then part of list (update)
// then empty list (delete)
func syncNamespacesDemo(clusterClient *client.Client) {
	// generate some namespaces
	namespaces := []string{}
	for i := 0; i < 100; i++ {
		namespaces = append(namespaces, fmt.Sprintf("namespace-%d", i))
	}

	// Create them
	err := clusterClient.SyncNamespaces(namespaces)
	if err != nil {
		glog.Fatalf("Failed to sync namespaces %v", err)
	}

	// Delete some of them
	err = clusterClient.SyncNamespaces(namespaces[50:])
	if err != nil {
		glog.Fatalf("Failed to sync namespaces %v", err)
	}

	// Delete all of them
	err = clusterClient.SyncNamespaces([]string{})
	if err != nil {
		glog.Fatalf("Failed to sync namespaces %v", err)
	}
}
