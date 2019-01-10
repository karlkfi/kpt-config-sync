/*
Copyright 2018 Google LLC.

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

package v1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client talks to k8s API server.
type Client interface {
	// Get retrieves an obj for the given object key from the Kubernetes Cluster.
	// obj must be a struct pointer so that obj can be updated with the response
	// returned by the Server.
	// This uses the same (non-Go-idiomatic) method signature as
	// sigs.k8s.io/controller-runtime/pkg/client, which keeps the code consistent
	// and allows Client to be extended later to implement the controller-runtime
	// client.
	Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error
}

// container is an interface that combines the things we need from TF Resource interface and the runtime.Object interface.
type container interface {
	ID() string
	GetObjectKind() schema.ObjectKind
	DeepCopyObject() runtime.Object
}

// ResourceID implements the ResourceClient interface
func ResourceID(ctx context.Context, client Client, Kind string, Name string) (string, error) {
	var res container
	resName := types.NamespacedName{Name: Name}
	switch Kind {
	case OrganizationKind:
		res = &Organization{}
	case FolderKind:
		res = &Folder{}
	case ProjectKind:
		res = &Project{}
	default:
		return "", fmt.Errorf("invalid kind: %v", Kind)
	}
	if err := client.Get(ctx, resName, res); err != nil {
		return "", errors.Wrapf(err, "failed to get resource: %v/%v", Kind, Name)
	}
	return res.ID(), nil
}
