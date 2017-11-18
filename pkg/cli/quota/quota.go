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
// objects.
package quota

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/cli"
	namespacewalker "github.com/google/stolos/pkg/client/namespace"
	"github.com/pkg/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

const (
	// Maximum number of quota items requested at once from the Kubernetes API server.
	resourceQuotaPageSizeItems = 100
)

func printForNamespace(
	namespace string, list *policyhierarchy.StolosResourceQuotaList) error {
	fmt.Printf("# Namespace: %q\n", namespace)
	fmt.Printf("#\n")
	e := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	return e.Encode(list, os.Stdout)
}

// GetHierarchical implements the 'kubectl plugin stolos get quota' command.
func GetHierarchical(ctx *cli.CommandContext, args []string) error {
	apiGroupClient := ctx.Client.Kubernetes().CoreV1()
	namespaces, _, err := namespacewalker.GetAncestry(
		apiGroupClient.Namespaces(), ctx.Namespace)
	if err != nil {
		return errors.Wrapf(err, "while getting ancestry for %q", ctx.Namespace)
	}
	// Now, get quota objects for everything.
	glog.V(5).Infof("Namespace hierarchy: %v", namespaces)

	policyHierarchyClient := ctx.Client.PolicyHierarchy().K8usV1()
	for _, ns := range namespaces {
		nsName := ns.ObjectMeta.Name
		quotaClient := policyHierarchyClient.StolosResourceQuotas(
			ns.ObjectMeta.Name)
		continueToken := ""
		for {
			result, err := quotaClient.List(meta.ListOptions{
				Limit:    resourceQuotaPageSizeItems,
				Continue: continueToken,
			})
			if err != nil {
				return errors.Wrapf(
					err, "while getting quota for namespace: %q", ns)
			}
			err = printForNamespace(nsName, result)
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
