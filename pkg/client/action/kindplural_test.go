package action

import (
	"fmt"
	"testing"
)

type pluralTestCase struct {
	input  string
	output string
}

func (s *pluralTestCase) Run(t *testing.T) {
	actual := Plural(s.input)
	if actual != s.output {
		t.Errorf("Expected %s, got %s", actual, s.output)
	}
}

var pluralTestCases = []pluralTestCase{
	pluralTestCase{
		input:  "PolicyNode",
		output: "PolicyNodes",
	},
	pluralTestCase{
		input:  "ClusterPolicy",
		output: "ClusterPolicies",
	},
}

func TestPlural(t *testing.T) {
	for _, testcase := range pluralTestCases {
		t.Run(fmt.Sprintf("%s->%s", testcase.input, testcase.output), testcase.Run)
	}
}
