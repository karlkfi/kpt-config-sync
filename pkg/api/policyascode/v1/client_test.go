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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// stubClient fakes a k8s API server client used in testing.
type stubClient struct {
	obj runtime.Object
}

// Get fakes the real client's Get() method by copying its own object content to obj according
// to the type of obj.
func (c *stubClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	switch obj.(type) {
	case *Organization:
		(c.obj.(*Organization)).DeepCopyInto(obj.(*Organization))
	case *Folder:
		(c.obj.(*Folder)).DeepCopyInto(obj.(*Folder))
	case *Project:
		(c.obj.(*Project)).DeepCopyInto(obj.(*Project))
	default:
		return fmt.Errorf("invalid runtime object type: %v", reflect.TypeOf(obj))
	}
	return nil
}
