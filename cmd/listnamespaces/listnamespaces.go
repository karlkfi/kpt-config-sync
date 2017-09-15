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
	"os"
	"os/signal"
	"syscall"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	flagSyncPolicyHierarchy = flag.Bool(
		"sync_policy_hierarchy", false, "demonstrate syncing policy hierarchy from custom resource")

	flagWatchPolicyHierarchy = flag.Bool(
		"watch_policy_hierarchy", false, "demonstrate watching policy hierarchy from custom resource")

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

	if *flagSyncPolicyHierarchy {
		syncPolicyHierarchyDemo(clusterClient)
	}

	if *flagWatchPolicyHierarchy {
		WatchSyncPolicyHierarchy(clusterClient)
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

func syncPolicyHierarchyDemo(clusterClient *client.Client) {
	err := clusterClient.SyncPolicyHierarchy()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to sync policy hierarchy"))
	}
}

// WatchSyncPolicyHierarchy will sync the namespaces then watch the policynodes custom resource
// for changes and sync namespaces as appropriate.
func WatchSyncPolicyHierarchy(clusterClient *client.Client) {
	policyNodes, resourceVersion, err := clusterClient.FetchPolicyHierarchy()
	if err != nil {
		panic(errors.Wrapf(err, "Failed to fetch policies"))
	}

	namespaces := client.ExtractNamespaces(policyNodes)

	err = clusterClient.SyncNamespaces(namespaces)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to sync namespaces"))
	}

	watchIface, err := clusterClient.PolicyHierarchy().K8usV1().PolicyNodes().Watch(
		meta_v1.ListOptions{ResourceVersion: resourceVersion})
	if err != nil {
		panic(errors.Wrapf(err, "Failed to watch policy hierarchy"))
	}

	go func() {
		glog.Infof("Watching changes to policynodes at %s", resourceVersion)
		resultChan := watchIface.ResultChan()
		for {
			select {
			case event, ok := <-resultChan:
				if !ok {
					glog.Info("Channel closed, exiting")
					return
				}
				node := event.Object.(*policyhierarchy_v1.PolicyNode)
				glog.Infof("Got event %s %s", event.Type, node.Spec.Name)

				namespace := client.ExtractNamespace(node)

				var action client.NamespaceAction
				switch event.Type {
				case watch.Added:
					// add the ns
					action = clusterClient.NamespaceCreateAction(namespace)
				case watch.Modified:
				case watch.Deleted:
					// delete the ns
					action = clusterClient.NamespaceDeleteAction(namespace)
				case watch.Error:
					panic(errors.Wrapf(err, "Got error during watch operation"))
				}

				if action == nil {
					continue
				}

				err := action.Execute()
				if err != nil {
					glog.Errorf("Failed to perform action %s on %s: %s", action.Operation(), action.Name(), err)
				}
			}
		}
	}()

	// wait for signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	glog.Info("Waiting for shutdown signal...")
	s := <-c
	glog.Info("Got signal %v, shutting down", s)
	watchIface.Stop()
}
