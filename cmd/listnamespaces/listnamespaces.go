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

// An example for usage of the client package.
package main

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client"
)

func main() {
	client, err := client.NewClient("minikube")
	if err != nil {
		glog.Fatalf("Error creating client: %v\n", err)
	}
	state, err := client.GetState()
	if err != nil {
		glog.Fatalf("Error fetching state: %s\n", err)
	}

	fmt.Printf("Found namespaces:\n")
	for _, namespace := range state.Namespaces {
		fmt.Printf(" %s\n", namespace)
	}
}
