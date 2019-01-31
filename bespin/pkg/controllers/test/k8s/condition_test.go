package k8s_test

import (
	"reflect"
	"runtime"
	"testing"

	v1 "github.com/google/nomos/bespin/pkg/api/bespin/v1"
	"github.com/google/nomos/bespin/pkg/controllers/test/k8s"
)

func TestEqualIgnoreTransitionTime(t *testing.T) {
	condition := v1.Condition{}
	cType := reflect.TypeOf(&condition).Elem()
	if cType.NumField() != 5 {
		t.Fatalf("number of fields in type '%v/%v' has increased, this test needs to be updated",
			cType.PkgPath(), cType.Name())
	}
	testCases := []struct {
		Name           string
		ConditionOne   v1.Condition
		ConditionTwo   v1.Condition
		ExpectedResult bool
	}{
		{
			Name:           "Equal structs",
			ConditionOne:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ConditionTwo:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ExpectedResult: true,
		},
		{
			Name:           "Different times, all other values equal",
			ConditionOne:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ConditionTwo:   v1.Condition{LastTransitionTime: "2018-11-09", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ExpectedResult: true,
		},
		{
			Name:           "Different message",
			ConditionOne:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ConditionTwo:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message2", Reason: "Reason", Status: "True", Type: "Ready"},
			ExpectedResult: false,
		},
		{
			Name:           "Different reason",
			ConditionOne:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ConditionTwo:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason2", Status: "True", Type: "Ready"},
			ExpectedResult: false,
		},
		{
			Name:           "Different status",
			ConditionOne:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ConditionTwo:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "False", Type: "Ready"},
			ExpectedResult: false,
		},
		{
			Name:           "Different type",
			ConditionOne:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready"},
			ConditionTwo:   v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True", Type: "Ready2"},
			ExpectedResult: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := k8s.ConditionsEqualIgnoreTransitionTime(tc.ConditionOne, tc.ConditionTwo)
			if result != tc.ExpectedResult {
				functionName := runtime.FuncForPC(reflect.ValueOf(k8s.ConditionsEqualIgnoreTransitionTime).Pointer()).Name()
				t.Errorf("unexpected result for '%v': got '%v', want '%v'", functionName, result, tc.ExpectedResult)
			}
		})
	}
}

func TestConditionSlicesEqual(t *testing.T) {
	c1 := v1.Condition{LastTransitionTime: "2018-11-08", Message: "Message", Reason: "Reason", Status: "True"}
	c1DifferentTransitionTime := v1.Condition{LastTransitionTime: "2018-11-09", Message: "Message", Reason: "Reason", Status: "True"}
	c2 := v1.Condition{LastTransitionTime: "2018-11-08", Message: "Different Message", Reason: "Reason", Status: "True"}
	testCases := []struct {
		Name           string
		ConditionsOne  []v1.Condition
		ConditionsTwo  []v1.Condition
		ExpectedResult bool
	}{
		{
			Name:           "Nil slices",
			ConditionsOne:  nil,
			ConditionsTwo:  nil,
			ExpectedResult: true,
		},
		{
			Name:           "Empty slices",
			ConditionsOne:  []v1.Condition{},
			ConditionsTwo:  []v1.Condition{},
			ExpectedResult: true,
		},
		{
			Name:           "Equal slices, size one",
			ConditionsOne:  []v1.Condition{c1},
			ConditionsTwo:  []v1.Condition{c1},
			ExpectedResult: true,
		},
		{
			Name:           "Equal slices, with different transition times, size one",
			ConditionsOne:  []v1.Condition{c1},
			ConditionsTwo:  []v1.Condition{c1DifferentTransitionTime},
			ExpectedResult: true,
		},
		{
			Name:           "Different slices, size one",
			ConditionsOne:  []v1.Condition{c1},
			ConditionsTwo:  []v1.Condition{c2},
			ExpectedResult: false,
		},
		{
			Name:           "Equal slices, size two",
			ConditionsOne:  []v1.Condition{c1, c2},
			ConditionsTwo:  []v1.Condition{c1, c2},
			ExpectedResult: true,
		},
		{
			Name:           "Equal slices, with different transition times, size two",
			ConditionsOne:  []v1.Condition{c1, c2},
			ConditionsTwo:  []v1.Condition{c1DifferentTransitionTime, c2},
			ExpectedResult: true,
		},
		{
			Name:           "Different slices, size two",
			ConditionsOne:  []v1.Condition{c1, c1},
			ConditionsTwo:  []v1.Condition{c1, c2},
			ExpectedResult: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := k8s.ConditionSlicesEqual(tc.ConditionsOne, tc.ConditionsTwo)
			if result != tc.ExpectedResult {
				functionName := runtime.FuncForPC(reflect.ValueOf(k8s.ConditionSlicesEqual).Pointer()).Name()
				t.Errorf("unexpected result for '%v': got '%v', want '%v'", functionName, result, tc.ExpectedResult)
			}
		})
	}
}
