package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const prefix = `// Code generated by cmd/gen-core-scoper/main.go. DO NOT EDIT

package discovery

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CoreScoper returns a Scoper with the scopes of all core Kubernetes and Nomos
// types defined. Use this instead of building a Scoper from the response to an
// APIServer when one is unavailable.
func CoreScoper() Scoper {
	return map[schema.GroupKind]IsNamespaced{
`

func main() {
	apiResources, err := ioutil.ReadFile("cmd/gen-core-scoper/api_resources.txt")
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(apiResources), "\n")
	sb := strings.Builder{}
	sb.WriteString(prefix)
	for _, line := range lines {
		if line == "" {
			continue
		}

		// fields is a slice of the non-whitespace substrings of the line.
		fields := strings.Fields(line)
		if len(fields) == 2 {
			// This is part of the core group, whose APIGroup is empty string so we have
			// to add it manually.
			fields = append([]string{""}, fields...)
		}
		// fields is now APIGroup, isNamespaced, Kind

		scope := "ClusterScope"
		if fields[1] == "true" {
			scope = "NamespaceScope"
		}

		sb.WriteString(fmt.Sprintf("\t\tschema.GroupKind{Group: %q, Kind: %q}: %s,\n", fields[0], fields[2], scope))
	}

	sb.WriteString(`	}
}
`)

	err = ioutil.WriteFile("pkg/util/discovery/core_scoper.generated.go", []byte(sb.String()), os.ModePerm)
	if err != nil {
		panic(err)
	}
}
