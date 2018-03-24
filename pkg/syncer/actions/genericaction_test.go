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
package actions

import (
	"testing"

	"github.com/google/nomos/pkg/client/action"
	actions_testing "github.com/google/nomos/pkg/syncer/actions/testing"
	"github.com/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGenericActionBase(t *testing.T) {
	testOperation := action.OperationType("testoperation")
	testResourceImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	g := genericActionBase{
		operation:         testOperation,
		resource:          testResourceImpl.ObjArg,
		resourceInterface: testResourceImpl,
	}

	if g.Name() != "testresourcename" {
		t.Errorf("Should have returned testresourcename for name")
	}
	if g.Resource() != "testresource" {
		t.Errorf("Should have returned testresource for resource")
	}
	if g.Namespace() != "default" {
		t.Errorf("Should have returned default for namespace")
	}
	if g.Operation() != testOperation {
		t.Errorf("Should have returned %s for operation", testOperation)
	}
	expectString := "group/v1/TestResource/default/testresourcename/testoperation"
	if g.String() != expectString {
		t.Errorf("Should have returned %s for string, got %s", expectString, g.String())
	}
}

func TestGenericUpsertActionCreate(t *testing.T) {
	testResourceImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	action := NewGenericUpsertAction(testResourceImpl.ObjArg, testResourceImpl)
	var err error

	err = action.create()
	if err != nil {
		t.Errorf("Expected success for create")
	}

	testResourceImpl.CreateErr = api_errors.NewAlreadyExists(schema.GroupResource{}, "testresource")
	err = action.create()
	if err != nil {
		t.Errorf("Expected success for create")
	}

	testResourceImpl.CreateErr = errors.Errorf("Other error")
	err = action.create()
	if err == nil {
		t.Errorf("Expected fail for create")
	}
}

func TestGenericUpsertActionUpsert(t *testing.T) {
	testResourceImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	action := NewGenericUpsertAction(testResourceImpl.ObjArg, testResourceImpl)
	var err error

	// get -> equal -> update
	testResourceImpl.EqualReturn = false
	if err = action.upsert(); err != nil {
		t.Errorf("Expected success for upsert")
	}

	// get -> equal -> update fail
	testResourceImpl.UpdateErr = errors.Errorf("Failure")
	if err = action.upsert(); err == nil {
		t.Errorf("Expected failure for upsert")
	}

	// get -> equal true
	testResourceImpl.EqualReturn = true
	if err = action.upsert(); err != nil {
		t.Errorf("Expected success for upsert")
	}

	// get error
	testResourceImpl.GetErr = errors.Errorf("error!")
	if err = action.upsert(); err == nil {
		t.Errorf("Expected fail for upsert")
	}

	// get error not found
	testResourceImpl.GetErr = api_errors.NewNotFound(schema.GroupResource{}, "testresource")
	if err = action.upsert(); err != nil {
		t.Errorf("Expected success for upsert")
	}
}

func TestGenericDeleteAction(t *testing.T) {
	testResourceImpl := actions_testing.NewTestResourceInterfaceImpl(t)
	action := NewGenericDeleteAction(testResourceImpl.ObjArg, testResourceImpl)
	var err error

	// get -> delete
	if err = action.Execute(); err != nil {
		t.Errorf("Expected success for delete")
	}

	// get -> delete (notfound)
	testResourceImpl.DeleteErr = api_errors.NewNotFound(schema.GroupResource{}, "testresource")
	if err = action.Execute(); err != nil {
		t.Errorf("Expected success for delete")
	}

	// get -> delete (other error)
	testResourceImpl.DeleteErr = errors.Errorf("Failure!")
	if err = action.Execute(); err == nil {
		t.Errorf("Expected failure for delete")
	}

	// get (notfound)
	testResourceImpl.GetErr = api_errors.NewNotFound(schema.GroupResource{}, "testresource")
	if err = action.Execute(); err != nil {
		t.Errorf("Expected success for delete")
	}

	// get (other error)
	testResourceImpl.GetErr = errors.Errorf("Failure!")
	if err = action.Execute(); err == nil {
		t.Errorf("Expected failure for delete")
	}
}
