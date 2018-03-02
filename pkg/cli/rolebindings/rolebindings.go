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

// Package rolebindings defines a Stolos CLI plugin that allows viewing
// hierarchical roles and rolebindings.  See package 'registration' for the
// plugin registration code.
package rolebindings

import (
	"os"

	"github.com/google/stolos/pkg/cli"
	"github.com/google/stolos/pkg/cli/namespaces"
	"github.com/google/stolos/pkg/cli/output"
	"github.com/pkg/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Load this many items at a time on a List operation.
const maxItemsPerPage = 100

// GetHierarchicalRoleBindings gets the hierarchical role bindings from the
// cluster.
func GetHierarchicalRoleBindings(ctx *cli.CommandContext, args []string) error {
	namespaces, _, err := namespaces.GetAncestryFromContext(ctx)
	if err != nil {
		return err
	}
	clientSet := ctx.Client.Kubernetes().RbacV1()
	for _, ns := range namespaces {
		nsName := ns.ObjectMeta.Name
		client := clientSet.RoleBindings(ns.ObjectMeta.Name)
		continueToken := ""
		for {
			result, innerErr := client.List(meta.ListOptions{
				Limit:    maxItemsPerPage,
				Continue: continueToken,
			})
			if innerErr != nil {
				err = errors.Wrapf(
					innerErr, "while getting rolebindings for namespace: %q", ns)
				break
			}
			innerErr = output.PrintForNamespace(nsName, result, os.Stdout)
			if innerErr != nil {
				err = errors.Wrapf(
					innerErr, "while encoding rolebindings for namespace: %q", ns)
				break
			}
			if result.Continue == "" {
				break
			}
			continueToken = result.Continue
		}
	}
	return err
}

// GetHierarchicalRoles gets the hierarchical roles from the cluster.
func GetHierarchicalRoles(ctx *cli.CommandContext, args []string) error {
	namespaces, _, err := namespaces.GetAncestryFromContext(ctx)
	if err != nil {
		return err
	}
	clientSet := ctx.Client.Kubernetes().RbacV1()
	for _, ns := range namespaces {
		nsName := ns.ObjectMeta.Name
		client := clientSet.Roles(ns.ObjectMeta.Name)
		continueToken := ""
		for {
			result, innerErr := client.List(meta.ListOptions{
				Limit:    maxItemsPerPage,
				Continue: continueToken,
			})
			if innerErr != nil {
				err = errors.Wrapf(
					innerErr, "while getting roles for namespace: %q", ns)
				break
			}
			innerErr = output.PrintForNamespace(nsName, result, os.Stdout)
			if innerErr != nil {
				err = errors.Wrapf(
					innerErr, "while encoding roles for namespace: %q", ns)
				break
			}
			if result.Continue == "" {
				break
			}
			continueToken = result.Continue
		}
	}
	return err
}
