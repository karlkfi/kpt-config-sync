package action

import (
	"fmt"
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type pluralTestCase struct {
	input       interface{}
	output      string
	outputLower string
}

func (s *pluralTestCase) Run(t *testing.T) {
	actual := Plural(s.input)
	if actual != s.output {
		t.Errorf("Expected %s, got %s", actual, s.output)
	}
	actualLower := LowerPlural(s.input)
	if actualLower != s.outputLower {
		t.Errorf("Expected %s, got %s", actualLower, s.outputLower)
	}
}

var pluralTestCases = []pluralTestCase{
	pluralTestCase{
		input:       v1.PolicyNode{},
		output:      "PolicyNodes",
		outputLower: "policynodes",
	},
	pluralTestCase{
		input:       v1.ClusterPolicy{},
		output:      "ClusterPolicies",
		outputLower: "clusterpolicies",
	},
	pluralTestCase{
		input:       rbacv1.ClusterRole{},
		output:      "ClusterRoles",
		outputLower: "clusterroles",
	},
	pluralTestCase{
		input:       &rbacv1.ClusterRole{},
		output:      "ClusterRoles",
		outputLower: "clusterroles",
	},
}

func TestPlural(t *testing.T) {
	for _, testcase := range pluralTestCases {
		t.Run(fmt.Sprintf("%s->%s", testcase.input, testcase.output), testcase.Run)
	}
}
