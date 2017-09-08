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

// Command line util to sync a text file representation of a namespace hierarchy
// to a Kubernetes cluster.
package main

import (
	"flag"
	"fmt"

	"github.com/mdruskin/kubernetes-enterprise-control/pkg/adapter"
)

func main() {
	filename := flag.String("f", "", "Filename for hierarchical namespace configuration")
	flag.Parse()

	nodes, err := adapter.Load(*filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Number of org units: ", len(nodes))
	fmt.Printf("Org units %+v\n", nodes)
}
