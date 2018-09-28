// Package main is a kubectl plugin that automates OIDC credential generation
// and use.
package main

import (
	"flag"

	"github.com/google/nomos/pkg/oidc"
)

func main() {
	flag.Parse()
	oidc.Execute()
}
