/*
Copyright 2018 The Nomos Authors.
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

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	nomosapischeme "github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/pkg/client/restconfig"
	genericcontroller "github.com/google/nomos/pkg/generic-syncer/controller"
	"github.com/google/nomos/pkg/util/log"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	flag.Parse()
	log.Setup()

	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Fatalf("Failed to create manager: %v", err)
	}

	// Set up Scheme for generic resources.
	clientSet := apis.NewForConfigOrDie(cfg)
	scheme := mgr.GetScheme()
	gvks, err := genericcontroller.RegisterGenericResources(scheme, clientSet)
	if err != nil {
		glog.Fatalf("Could not register Generic Resources: %v", err)
	}

	// Set up Scheme for nomos resources.
	nomosapischeme.AddToScheme(scheme)

	// Set up all Controllers.
	if err := genericcontroller.AddPolicyNode(mgr, clientSet, gvks); err != nil {
		glog.Fatalf("Could not create PolicyNode controllers: %v", err)
	}
	if err := genericcontroller.AddClusterPolicy(mgr, clientSet, gvks); err != nil {
		glog.Fatalf("Could not create ClusterPolicy controllers: %v", err)
	}

	// Start the Managers.
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatalf("Error starting controller: %v", err)
	}
}
