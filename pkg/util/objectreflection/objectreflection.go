/*
Copyright 2018 The Stolos Authors.
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

package objectreflection

import (
	"fmt"
	"reflect"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Meta returns mutable TypeMeta and ObjectMeta fields from a runtime.Object.
func Meta(obj runtime.Object) (*meta_v1.TypeMeta, *meta_v1.ObjectMeta) {
	objValue := reflect.ValueOf(obj)
	if objValue.Type().Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Invalid type passed to Meta: %v", obj))
	}
	if objValue.IsNil() {
		panic(fmt.Sprintf("Nil object passed to Meta"))
	}

	objElem := objValue.Elem()                                    // Dereference the pointer
	typeMetaValue := objElem.FieldByName("TypeMeta")              // Get the TypeMeta field
	objectMetaValue := objElem.FieldByName("ObjectMeta")          // Get the ObjectMeta field
	typeMetaPtr := typeMetaValue.Addr()                           // Get a pointer to the TypeMeta field
	objectMetaPtr := objectMetaValue.Addr()                       // Get a pointer to the ObjectMeta field
	typeMeta := typeMetaPtr.Interface().(*meta_v1.TypeMeta)       // Cast the interface to a TypeMeta pointer
	objectMeta := objectMetaPtr.Interface().(*meta_v1.ObjectMeta) // Cast the interface to an ObjectMeta pointer
	return typeMeta, objectMeta
}

// GetNamespaceAndName returns the namespace and name from a runtime.Object.
func GetNamespaceAndName(obj runtime.Object) (string, string) {
	_, objectMeta := Meta(obj)
	return objectMeta.Namespace, objectMeta.Name
}
