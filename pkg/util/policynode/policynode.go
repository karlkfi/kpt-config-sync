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

package policynode

import (
	"strconv"

	"github.com/pkg/errors"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewPolicyNode creates a PolicyNode from the given spec and name.
func NewPolicyNode(name string, spec *policyhierarchy_v1.PolicyNodeSpec) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "PolicyNode",
			APIVersion: policyhierarchy_v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

// NewClusterPolicy creates a PolicyNode from the given spec and name.
func NewClusterPolicy(name string, spec *policyhierarchy_v1.ClusterPolicySpec) *policyhierarchy_v1.ClusterPolicy {
	return &policyhierarchy_v1.ClusterPolicy{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "ClusterPolicy",
			APIVersion: policyhierarchy_v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}
}

// GetResourceVersion parses the resource version string into an int64
func GetResourceVersion(node *policyhierarchy_v1.PolicyNode) (int64, error) {
	resourceVersionStr := node.ResourceVersion
	if resourceVersionStr == "" {
		return 0, errors.Errorf("Empty resource version in %#v", node)
	}

	resourceVersion, err := strconv.ParseInt(resourceVersionStr, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to parse resource version from %#v", node)
	}

	return resourceVersion, nil
}

// GetResourceVersionOrDie parses the resource version into an int and panics if there is an error.
func GetResourceVersionOrDie(node *policyhierarchy_v1.PolicyNode) int64 {
	resourceVersion, err := GetResourceVersion(node)
	if err != nil {
		panic(err)
	}
	return resourceVersion
}
