/*
Copyright 2017 The Nomos Authors.
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

package namespaces

import (
	"github.com/golang/glog"
	apipolicyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/cli"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	client_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetAncestry returns the ancestors of the given namespace by following the
// hierarchy using nomos parent label set on a namespace.  The first element
// of the returned slice represents the given namespace itself, second element
// is the parent, and so on. The last element represents the closest namespace
// to the root that the caller is authorized to get.  includesRoot indicates
// whether the returned namespaces includes the root namespace. This is false
// if the user is not authorized to get namespaces above a certain level in the
// hierarchy.  If the user is not authorized to get the given namespace itself,
// error is returned.
func GetAncestry(client client_v1.NamespaceInterface, namespace string) (
	namespaces []*core_v1.Namespace, includesRoot bool, error error) {
	nsAncestry := make([]*core_v1.Namespace, 0)

	for {
		ns, err := client.Get(namespace, meta.GetOptions{})
		if err != nil {
			if api_errors.IsForbidden(err) {
				if len(nsAncestry) == 0 {
					return nil, false, err
				}
				return nsAncestry, false, nil
			}
			return nil, false, errors.Wrapf(err, "Failed to get namespace %q", namespace)
		}
		nsAncestry = append(nsAncestry, ns)

		parent, exists := ns.Labels[apipolicyhierarchy.ParentLabelKey]
		if !exists {
			return nil, false, errors.Errorf("Parent label not set on namespace: %q.", namespace)
		}

		if parent == apipolicyhierarchy.NoParentNamespace {
			return nsAncestry, true, nil
		}

		namespace = parent
	}
}

// GetAncestryFromContext returns namespace ancestry as suppplied in the cli
// context.
func GetAncestryFromContext(ctx *cli.CommandContext) (
	[]*core_v1.Namespace, bool, error) {
	apiGroupClient := ctx.Client.Kubernetes().CoreV1()
	namespaces, isRoot, err := GetAncestry(
		apiGroupClient.Namespaces(), ctx.Namespace)
	glog.V(5).Infof("Namespace hierarchy: %+v", namespaces)
	return namespaces, isRoot, err
}
