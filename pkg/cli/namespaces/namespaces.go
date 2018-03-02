/*
Copyright 2017 The Stolos Authors.

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

// Package namespaces defines a Stolos CLI functionality that allows viewing namespace hierarchy.
package namespaces

import (
	"fmt"
	"strings"

	"github.com/google/stolos/pkg/cli"
)

func GetHierarchicalNamespaces(ctx *cli.CommandContext, args []string) error {
	namespaces, includesRoot, err := GetAncestry(
		ctx.Client.Kubernetes().CoreV1().Namespaces(), ctx.Namespace)
	if err != nil {
		return err
	}

	names := make([]string, 0)
	for _, ns := range namespaces {
		names = append(names, ns.Name)
	}
	if !includesRoot {
		names = append(names, "...")
	}

	fmt.Println(strings.Join(names, " > "))
	return nil
}
