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

// Demo for using the policy node CRD informer.
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/client/restconfig"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/cache"
	"github.com/google/stolos/pkg/client/informers/externalversions"
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

	// Create the informer
	informerFactory := externalversions.NewSharedInformerFactory(client.PolicyHierarchy(), time.Minute)
	informer := informerFactory.K8us().V1().PolicyNodes().Informer()

	informerFactory.Start(nil)

	fmt.Print("Wait for initial sync ... ")
	if !cache.WaitForCacheSync(
		nil,
		informer.HasSynced) {
		fmt.Print("timed out waiting for cache sync")
		return
	}
	fmt.Println("Synced")

	// Sample handler to add to the informer.
	handler := cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			crd := obj.(*v1.PolicyNode)
			fmt.Println(crd.Name, " Deleted")
		},
	}
	informer.AddEventHandler(handler)

	// Get a specific PolicyNode
	policyNode1, exists, err := informer.GetStore().GetByKey("policy-node-ns1")
	if exists {
		fmt.Printf("PolicyNode retrieved %v\n", policyNode1)
	} else {
		fmt.Println("PolicyNode1 not found")
	}
	// Print all policy nodes every 5 seconds
	for {
		<-time.After(5 * time.Second)
		go printAll(informer.GetStore())
	}
}

func printAll(store cache.Store) {
	fmt.Println("============= Listing all CRDs =========\n")
	all := store.List()
	for _, each := range all {
		crd := each.(*v1.PolicyNode)
		fmt.Printf("- %v\n", crd.Name)
	}
}
