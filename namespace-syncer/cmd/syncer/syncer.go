package main

import (
	"flag"
	"fmt"
	"github.com/mdruskin/kubernetes-enterprise-control/namespace-syncer/pkg/adapter"
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
