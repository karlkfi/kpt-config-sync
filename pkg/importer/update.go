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

package importer

import "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"

// Op represents an update operation type enum
type Op string

// Values for update operations
const (
	CreatePolicy = Op("Create")
	DeletePolicy = Op("Delete")
	UpdatePolicy = Op("Update")
)

// Update encapsulates info required for updating the PolicyNode CR
type Update struct {
	Operation  Op
	PolicyNode *v1.PolicyNode
}

// NewUpdate creates a new update.
func NewUpdate(op Op, policyNode *v1.PolicyNode) *Update {
	return &Update{Operation: op, PolicyNode: policyNode}
}
