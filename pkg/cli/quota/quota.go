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

// Package quota defines a Stolos CLI plugin that allows viewing Stolos quota
// objects.  See package 'registration' for the plugin registration code.
package quota

import (
	"os"

	"github.com/google/stolos/pkg/cli"
	"github.com/google/stolos/pkg/cli/output"
	ns "github.com/google/stolos/pkg/client/namespace"
	"github.com/pkg/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const maxItemsPerPage = 100

// GetHierarchical implements the 'kubectl plugin stolos get quota' command.
func GetHierarchical(ctx *cli.CommandContext, args []string) error {
	namespaces, _, err := ns.GetAncestryFromContext(ctx)
	if err != nil {
		return err
	}
	clientSet := ctx.Client.PolicyHierarchy().K8usV1()
	for _, ns := range namespaces {
		nsName := ns.ObjectMeta.Name
		client := clientSet.StolosResourceQuotas(ns.ObjectMeta.Name)
		continueToken := ""
		for {
			result, err := client.List(meta.ListOptions{
				Limit:    maxItemsPerPage,
				Continue: continueToken,
			})
			if err != nil {
				return errors.Wrapf(
					err, "while getting quota for namespace: %q", ns)
			}
			err = output.PrintForNamespace(nsName, result, os.Stdout)
			if err != nil {
				return errors.Wrapf(
					err, "while encoding quota for namespace: %q", ns)
			}
			if result.Continue == "" {
				break
			}
			continueToken = result.Continue
		}
	}
	return nil
}
