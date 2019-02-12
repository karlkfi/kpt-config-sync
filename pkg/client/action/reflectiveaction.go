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
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OperationType indicates which type of operation the reflective action will perform.
type OperationType string

const (
	// CreateOperation will create the resource and fail if it exists.
	CreateOperation = OperationType("create")
	// UpsertOperation will create or update the resource
	UpsertOperation = OperationType("upsert")
	// UpdateOperation will attempt to update the resource until it succeeds or gives up
	UpdateOperation = OperationType("update")
	// DeleteOperation will delete the resource
	DeleteOperation = OperationType("delete")
)

// noUpdateNeededError is returned if no update is needed for the given resource.
type noUpdateNeededError struct {
}

// Error implements error
func (e *noUpdateNeededError) Error() string {
	return "noUpdateNeededError"
}

// NoUpdateNeeded returns an error code for update not required.
func NoUpdateNeeded() error {
	return &noUpdateNeededError{}
}

// IsNoUpdateNeeded checks for whether the returned error is noUpdateNeededError
func IsNoUpdateNeeded(err error) bool {
	_, ok := err.(*noUpdateNeededError)
	return ok
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

// Item returns the resource which is being modified for the case of an upsert action.
func (s *ReflectiveActionBase) Item() runtime.Object {
	return s.resource
}

// Resource implements Interface
func (s *ReflectiveActionBase) Resource() string {
	return s.spec.Resource
}

// Kind implements Interface
func (s *ReflectiveActionBase) Kind() string {
	return s.spec.KindPlural
}

// Namespace implements Interface
func (s *ReflectiveActionBase) Namespace() string {
	return s.namespace
}

// Group implements Interface
func (s *ReflectiveActionBase) Group() string {
	return s.spec.GroupVersion.Group
}

// Version implements Interface
func (s *ReflectiveActionBase) Version() string {
	return s.spec.GroupVersion.Version
}

// Name implements Interface
func (s *ReflectiveActionBase) Name() string {
	return s.name
}

// Operation implements Interface
func (s *ReflectiveActionBase) Operation() OperationType {
	return s.operation
}

// String implements Interface
func (s *ReflectiveActionBase) String() string {
	var fields []string
	if g := s.Group(); g != "" {
		fields = append(fields, g)
	}
	fields = append(fields, s.Version(), s.Kind())
	if ns := s.Namespace(); ns != "" {
		fields = append(fields, ns)
	}
	fields = append(fields, s.Name(), string(s.Operation()))

	return strings.Join(fields, "/")
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

// listerGet gets the resource via the lister.
// Example of what this is effectively doing:
// lister := kubernetesInformerFactory.Rbac().V1().ClusterRoles().Lister() // first line
// return lister.Get(s.name)
func (s *ReflectiveActionBase) listerGet() (runtime.Object, error) {
	lister := s.spec.listerValue(s.namespace)
	getMethod := lister.MethodByName("Get")
	return s.toObjectError(getMethod.Call([]reflect.Value{reflect.ValueOf(s.name)}))
}

// delete deletes the resource using the client
// Example of what this is effectively doing:
// client := kubernetesClient.RbacV1().ClusterRoles() // first line
// return client.Delete("foo", &metav1.DeleteOptions{})
func (s *ReflectiveActionBase) delete() error {
	client := s.client()
	deleteMethod := client.MethodByName("Delete")
	deleteArgs := []reflect.Value{
		reflect.ValueOf(s.name),
		reflect.ValueOf(&metav1.DeleteOptions{}),
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

// tryUpdate will attempt to update on the current object.
// Example code:
// client := kubernetesClient.RbacV1().ClusterRoles() // first line
// return client.Update(obj)
func (s *ReflectiveActionBase) tryUpdate(obj runtime.Object) (runtime.Object, error) {
	client := s.client()
	updateMethod := client.MethodByName("Update")
	updateObject := reflect.ValueOf(obj)
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

// ReflectiveCreateAction implements a create action for all generated client stubs.  This will not
// attempt to modify the action if it already exists.
type ReflectiveCreateAction struct {
	ReflectiveActionBase
}

var _ Interface = &ReflectiveCreateAction{}

// Execute implements Interface
func (s *ReflectiveCreateAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	Actions.WithLabelValues(s.Resource(), string(s.Operation())).Inc()
	APICalls.WithLabelValues(s.Resource(), string(s.Operation())).Inc()
	timer := prometheus.NewTimer(APICallDuration.WithLabelValues(s.Resource(), string(s.Operation())))
	defer timer.ObserveDuration()
	if _, err := s.create(); err != nil {
		return errors.Wrapf(err, "failed to execute %s", s)
	}
	return nil
}

// NewReflectiveCreateAction creates a new create action given a namespace, name and spec. Note that
// for cluster level resources namespace MUST be the empty string.
func NewReflectiveCreateAction(
	namespace, name string, resource runtime.Object, spec *ReflectiveActionSpec) *ReflectiveCreateAction {
	return &ReflectiveCreateAction{
		ReflectiveActionBase: ReflectiveActionBase{
			namespace: namespace,
			name:      name,
			operation: CreateOperation,
			resource:  resource,
			spec:      spec,
		},
	}
}

// Update is a function that updates the state of an API object. The argument is expected to be a copy of the object,
// so no there is no need to worry about mutating the argument when implementing an Update function.
type Update func(runtime.Object) (runtime.Object, error)

// ReflectiveUpdateAction implements an update action for all generated client stubs.
type ReflectiveUpdateAction struct {
	ReflectiveActionBase
	update   Update
	MaxTries int
}

var _ Interface = &ReflectiveUpdateAction{}

// NewReflectiveUpdateAction creates a new update action given a namespace, name and spec.  The
// action will retry until the update succeeds or update returns error.
func NewReflectiveUpdateAction(
	namespace, name string, update Update, spec *ReflectiveActionSpec) *ReflectiveUpdateAction {
	return &ReflectiveUpdateAction{
		ReflectiveActionBase: ReflectiveActionBase{
			namespace: namespace,
			name:      name,
			operation: UpdateOperation,
			spec:      spec,
		},
		update:   update,
		MaxTries: 5,
	}
}

// Execute implements Interface
func (s *ReflectiveUpdateAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	Actions.WithLabelValues(s.Resource(), string(s.Operation())).Inc()
	return s.doUpdate()
}

func (s *ReflectiveUpdateAction) doUpdate() error {
	var err error
	for tryNum := 0; tryNum < s.MaxTries; tryNum++ {
		var obj, newObj runtime.Object
		obj, err = s.listerGet()
		if err != nil {
			return err
		}

		newObj, err = s.update(obj.DeepCopyObject())
		if err != nil {
			if IsNoUpdateNeeded(err) {
				return nil
			}
			return err
		}

		APICalls.WithLabelValues(s.Resource(), string(s.Operation())).Inc()
		timer := prometheus.NewTimer(APICallDuration.WithLabelValues(s.Resource(), string(s.Operation())))
		_, err = s.tryUpdate(newObj)
		timer.ObserveDuration()
		if err == nil {
			glog.V(1).Infof("OK: %s", s)
			return nil
		}
		if !apierrors.IsConflict(err) {
			return err
		}
	}
	return errors.Wrapf(err, "max update tries exceeded in %s", s)
}

// UpdatedResource returns a preview of what the updated resource will look like.
func (s *ReflectiveUpdateAction) UpdatedResource(obj runtime.Object) (runtime.Object, error) {
	return s.update(obj)
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

// UpsertedResouce returns the resource that will be upserted.
func (s *ReflectiveUpsertAction) UpsertedResouce() runtime.Object {
	return s.resource
}

// Execute implements Interface
func (s *ReflectiveUpsertAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	Actions.WithLabelValues(s.Resource(), string(s.Operation())).Inc()
	return s.doUpsert()
}

func (s *ReflectiveActionBase) doCreate() error {
	APICalls.WithLabelValues(s.Resource(), "create").Inc()
	timer := prometheus.NewTimer(APICallDuration.WithLabelValues(s.Resource(), "create"))
	_, err := s.create()
	timer.ObserveDuration()
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return s.doUpsert()
		}
		return errors.Wrapf(err, "failed during create for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

func (s *ReflectiveActionBase) doUpsert() error {
	resource, err := s.listerGet()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return s.doCreate()
		}
		return errors.Wrapf(err, "failed to get resource for %s", s)
	}

	if s.spec.Equal(s.resource, resource) {
		return nil
	}

	APICalls.WithLabelValues(s.Resource(), "update").Inc()
	timer := prometheus.NewTimer(APICallDuration.WithLabelValues(s.Resource(), "update"))
	defer timer.ObserveDuration()
	if _, err = s.update(resource); err != nil {
		return errors.Wrapf(err, "failed to update for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

// ReflectiveDeleteAction implements a delete action for all generated client stubs.
type ReflectiveDeleteAction struct {
	ReflectiveActionBase
	timeout time.Duration
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

// NewBlockingReflectiveDeleteAction creates a new delete action given a namespace, name and spec. Note that
// for cluster level resources namespace MUST be the empty string.
func NewBlockingReflectiveDeleteAction(
	namespace, name string, timeout time.Duration, spec *ReflectiveActionSpec) *ReflectiveDeleteAction {
	return &ReflectiveDeleteAction{
		ReflectiveActionBase: ReflectiveActionBase{
			namespace: namespace,
			name:      name,
			operation: DeleteOperation,
			spec:      spec,
		},
		timeout: timeout,
	}
}

// Execute implements Interface
func (s *ReflectiveDeleteAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	Actions.WithLabelValues(s.Resource(), string(s.Operation())).Inc()

	o, err := s.listerGet()
	if err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(5).Infof("not found during lister get %s", s)
			return nil
		}
		return errors.Wrapf(err, "get failed for %s", s)
	}

	m, ok := o.(metav1.Object)
	if !ok {
		panic(fmt.Sprintf("programmer error, attempting to delete object with no metadata field: %v", m))
	}
	if IsFinalizing(m) {
		glog.V(1).Infof("attempting to delete an object that is finalizing: %s", s)
		return nil
	}

	APICalls.WithLabelValues(s.Resource(), "delete").Inc()
	timer := prometheus.NewTimer(APICallDuration.WithLabelValues(s.Resource(), string(s.Operation())))
	defer timer.ObserveDuration()
	err = s.delete()
	if err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(5).Infof("not found during delete %s", s)
			return nil
		}
		return errors.Wrapf(err, "delete failed for %s", s)
	}

	if err := s.waitForDelete(); err != nil {
		return err
	}

	glog.V(1).Infof("OK: %s", s)
	return nil
}

func (s *ReflectiveDeleteAction) waitForDelete() error {
	if s.timeout == 0 {
		return nil
	}

	deadline := time.Now().Add(s.timeout)
	for time.Now().Before(deadline) {
		_, err := s.listerGet()
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return errors.Wrapf(err, "failed to list object while waiting for delete")
		}
		time.Sleep(time.Millisecond * 25)
	}
	return errors.Errorf("%s deadline exceeded (%s)", s, s.timeout)
}
