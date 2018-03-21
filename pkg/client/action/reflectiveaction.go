/*
Copyright 2018 The Nomos Authors.
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

package action

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OperationType indicates which type of operation the reflective action will perform.
type OperationType string

const (
	// UpsertOperation will create or update the resource
	UpsertOperation = OperationType("upsert")
	// DeleteOperation will delete the resource
	DeleteOperation = OperationType("delete")
)

// isSubset returns true if subset is a subset of set.  If subset is emtpy, it is always considered
// a subset of set even if set is empty.  Subset is defined as follows: A is a subset of B if
// for all (k, v) in A, key k exists in B and key k's value in b is v.
func isSubset(set map[string]string, subset map[string]string) bool {
	if len(subset) == 0 {
		return true
	}
	if len(set) == 0 {
		// set empty, subset has items
		return false
	}

	// set and subset have items
	for k, v := range subset {
		if setValue, found := set[k]; !found || v != setValue {
			return false
		}
	}
	return true
}

// MetaSubset returns true if subset's labels and annotations are a subset of the labels and
// annotations in set.  See isSubset for the definition of subset for labels / annotations.
func MetaSubset(set meta_v1.ObjectMeta, subset meta_v1.ObjectMeta) bool {
	return isSubset(set.Labels, subset.Labels) && isSubset(set.Annotations, subset.Annotations)
}

// ObjectMetaSubset returns true if the Meta field of subset is a subset of the meta field for set.
func ObjectMetaSubset(set runtime.Object, subset runtime.Object) bool {
	setMeta := reflect.ValueOf(set).Elem().FieldByName("ObjectMeta").Interface().(meta_v1.ObjectMeta)
	subsetMeta := reflect.ValueOf(subset).Elem().FieldByName("ObjectMeta").Interface().(meta_v1.ObjectMeta)
	if !MetaSubset(setMeta, subsetMeta) {
		return false
	}
	return true
}

// ReflectiveActionSpec represents information and objects needed for performing actions on a given
// GroupVersionKind.
type ReflectiveActionSpec struct {
	// The plural of a kind, eg, Roles, RoleBindings, Policies as used for getting the client from
	// the generated code.
	KindPlural string
	// EqualSpec is the per-Kind equal operation that check for equality for the spec of an object.
	// Meta fields (ObjectMeta and TypeMeta) equality is done automatically and need not be done
	// by this function.
	EqualSpec func(lhs runtime.Object, rhs runtime.Object) bool
	// Client is the client-gen generated stub for the given API group, for example:
	// kubernetesClient.RbacV1() or kubernetesClient.CoreV1()
	Client interface{}
	// Lister is the lister from the generated informer for the given, example:
	// kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister()
	Lister interface{}
}

// Equal returns true if the two objects have equivalent per-kind spec equality and the
// labels and annotations are a superset of the declared labels and annotations.
func (s ReflectiveActionSpec) Equal(declared runtime.Object, actual runtime.Object) bool {
	if !s.EqualSpec(actual, declared) {
		return false
	}
	return ObjectMetaSubset(actual, declared)
}

// ReflectiveActionBase is the base implementation for performing actions using reflection. This
// supports both namespace and cluster scoped resources.
type ReflectiveActionBase struct {
	namespace string                // The namespace, leave blank for cluster scoped resources
	name      string                // The name of the resource
	operation OperationType         // The type operation that the action will perform
	resource  runtime.Object        // The resource itself, will not be set for delete actions.
	spec      *ReflectiveActionSpec // Definitions for type specific behavior and functionality
}

// Resource implements Interface
func (s *ReflectiveActionBase) Resource() string {
	return s.spec.KindPlural
}

// Namespace implements Interface
func (s *ReflectiveActionBase) Namespace() string {
	return s.namespace
}

// Name returns the name of the resource
func (s *ReflectiveActionBase) Name() string {
	return s.name
}

// Operation implements Interface
func (s *ReflectiveActionBase) Operation() string {
	return string(s.operation)
}

// String implements Interface
func (s *ReflectiveActionBase) String() string {
	if s.Namespace() != "" {
		return fmt.Sprintf(
			"%s.%s.%s.%s",
			s.Resource(),
			s.Namespace(),
			s.Name(),
			s.Operation())
	}
	return fmt.Sprintf(
		"%s.%s.%s",
		s.Resource(),
		s.Name(),
		s.Operation())
}

// client returns the appropriate client taking into account cluster / namespace scoping
// Example of what this is effectively doing:
// -- Definition for s.spec.Client
// s.spec.Client := kubernetesClient.RbacV1()
// -- For cluster scoped resources --
// return s.spec.Client.ClusterRoles()
// -- For namespace scoped resources --
// return s.spec.Client.Roles(s.namespace)
func (s *ReflectiveActionBase) client() reflect.Value {
	groupVersionClient := reflect.ValueOf(s.spec.Client)
	getKindClientMethod := groupVersionClient.MethodByName(s.spec.KindPlural)

	var kindClientArgs []reflect.Value
	if s.namespace != "" {
		kindClientArgs = []reflect.Value{reflect.ValueOf(s.namespace)}
	}

	getKindClientReturns := getKindClientMethod.Call(kindClientArgs)
	if len(getKindClientReturns) != 1 {
		panic(fmt.Sprintf("getKindClientMethod returned invalid number of args %v", getKindClientReturns))
	}

	return getKindClientReturns[0]
}

// lister returns the appropraite lister taking into account cluster / namespace scoping
// Example of what this is effectively doing:
// -- For cluster scoped resources --
// s.spec.Lister := kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister()
// return s.spec.Lister
// -- For namesapce scoped resources --
// s.spec.Lister := kubernetesInformerFactory.Rbac().V1().Roles().Lister()
// return s.spec.Lister.Roles(s.namespace)
func (s *ReflectiveActionBase) lister() reflect.Value {
	listerValue := reflect.ValueOf(s.spec.Lister)
	if s.namespace == "" {
		return listerValue
	}

	methodValue := listerValue.MethodByName(s.spec.KindPlural)
	listerReturnValues := methodValue.Call([]reflect.Value{reflect.ValueOf(s.namespace)})
	if len(listerReturnValues) != 1 {
		panic(fmt.Sprintf("Getting lister returned invalid number of values"))
	}
	return listerReturnValues[0]
}

// listerGet gets the resource via the lister.
// Example of what this is effectively doing:
// lister := kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister() // first line
// return lister.Get(s.name)
func (s *ReflectiveActionBase) listerGet() (runtime.Object, error) {
	lister := s.lister()
	getMethod := lister.MethodByName("Get")
	return s.toObjectError(getMethod.Call([]reflect.Value{reflect.ValueOf(s.name)}))
}

// delete deletes the resource using the client
// Example of what this is effectively doing:
// client := kubernetesClient.RbacV1().ClusterRoles() // first line
// return client.Delete("foo", &meta_v1.DeleteOptions{})
func (s *ReflectiveActionBase) delete() error {
	client := s.client()
	deleteMethod := client.MethodByName("Delete")
	deleteArgs := []reflect.Value{
		reflect.ValueOf(s.name),
		reflect.ValueOf(&meta_v1.DeleteOptions{}),
	}
	deleteReturns := deleteMethod.Call(deleteArgs)
	if len(deleteReturns) != 1 {
		panic(fmt.Sprintf("deleteReturns returned invalid number of args %v", deleteReturns))
	}
	if deleteReturns[0].IsNil() {
		return nil
	}
	return deleteReturns[0].Interface().(error)
}

// create creates the resource using the client
// Example of what this is effectively doing:
// client := kubernetesClient.RbacV1().ClusterRoles() // first line
// return client.Create(s.resource)
func (s *ReflectiveActionBase) create() (runtime.Object, error) {
	client := s.client()
	createMethod := client.MethodByName("Create")
	createArgs := []reflect.Value{reflect.ValueOf(s.resource)}
	return s.toObjectError(createMethod.Call(createArgs))
}

// update will udpate the resource using the client
// Example of what this is effectively doing:
// client := kubernetesClient.RbacV1().ClusterRoles() // first line
// objCopy := s.resource.DeepCopy()
// objCopy.ResourceVersion = currentResource.ResourceVersion
// return client.Update(objCopy)
func (s *ReflectiveActionBase) update(currentResource runtime.Object) (runtime.Object, error) {
	client := s.client()
	updateMethod := client.MethodByName("Update")
	resourceVersion := reflect.ValueOf(currentResource).Elem().
		FieldByName("ObjectMeta").FieldByName("ResourceVersion").Interface().(string)

	updateObject := reflect.ValueOf(s.resource.DeepCopyObject())
	updateObject.Elem().FieldByName("ObjectMeta").FieldByName("ResourceVersion").SetString(resourceVersion)
	updateArgs := []reflect.Value{updateObject}
	return s.toObjectError(updateMethod.Call(updateArgs))
}

// toObjectError takes a [runtime.Object, error] in the form of their reflect.Value representation
// and appropriately converts to (runtime.Object, error)
func (s ReflectiveActionBase) toObjectError(returnValues []reflect.Value) (runtime.Object, error) {
	if len(returnValues) != 2 {
		panic(fmt.Sprintf("values call returned invalid number of args %v", returnValues))
	}

	if returnValues[0].IsNil() {
		return nil, returnValues[1].Interface().(error)
	}

	retObject := returnValues[0].Interface().(runtime.Object)
	if returnValues[1].IsNil() {
		return retObject, nil
	}

	retError := returnValues[1].Interface().(error)
	return retObject, retError
}

// ReflectiveUpsertAction implements an upsert action for all generated client stubs.
type ReflectiveUpsertAction struct {
	ReflectiveActionBase
}

var _ Interface = &ReflectiveUpsertAction{}

// NewReflectiveUpsertAction creates a new upsert action given a namespace, name and spec. Note that
// for cluster level resources namespace MUST be the empty string.
func NewReflectiveUpsertAction(
	namespace, name string, resource runtime.Object, spec *ReflectiveActionSpec) *ReflectiveUpsertAction {
	return &ReflectiveUpsertAction{
		ReflectiveActionBase: ReflectiveActionBase{
			namespace: namespace,
			name:      name,
			operation: UpsertOperation,
			resource:  resource,
			spec:      spec,
		},
	}
}

// Execute implements Interface
func (s *ReflectiveUpsertAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	return s.doUpsert()
}

func (s *ReflectiveActionBase) doCreate() error {
	if _, err := s.create(); err != nil {
		if api_errors.IsAlreadyExists(err) {
			return s.doUpsert()
		}
		return errors.Wrapf(err, "failed during create for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

func (s *ReflectiveActionBase) doUpsert() error {
	resouce, err := s.listerGet()
	if err != nil {
		if api_errors.IsNotFound(err) {
			return s.doCreate()
		}
		return errors.Wrapf(err, "failed to get resource for %s", s)
	}

	if s.spec.Equal(s.resource, resouce) {
		return nil
	}

	if _, err = s.update(resouce); err != nil {
		return errors.Wrapf(err, "failed to update for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

// ReflectiveDeleteAction implements an upsert action for all generated client stubs.
type ReflectiveDeleteAction struct {
	ReflectiveActionBase
}

var _ Interface = &ReflectiveDeleteAction{}

// NewReflectiveDeleteAction creates a new delete action given a namespace, name and spec. Note that
// for cluster level resources namespace MUST be the empty string.
func NewReflectiveDeleteAction(
	namespace, name string, spec *ReflectiveActionSpec) *ReflectiveDeleteAction {
	return &ReflectiveDeleteAction{
		ReflectiveActionBase: ReflectiveActionBase{
			namespace: namespace,
			name:      name,
			operation: DeleteOperation,
			spec:      spec,
		},
	}
}

// Execute implements Interface
func (s *ReflectiveDeleteAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)

	_, err := s.listerGet()
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "get failed for %s", s)
	}

	err = s.delete()
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "delete failed for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}
