package filesystem

import (
	"testing"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

type validatorsTestCase struct {
	testName      string
	v             *validator
	expectedError bool
}

var namespaceType = meta_v1.TypeMeta{Kind: "Namespace", APIVersion: "v1"}

var testCases = []validatorsTestCase{
	{"HasName valid", newValidator().HasName(&resource.Info{Name: "foo"}, "foo"), false},
	{"HasName invalid", newValidator().HasName(&resource.Info{Name: "foo"}, "bar"), true},
	{"HasNamespace valid", newValidator().HasNamespace(&resource.Info{Namespace: "foo"}, "foo"), false},
	{"HasNamespace invalid", newValidator().HasNamespace(&resource.Info{Namespace: "foo"}, "bar"), true},
	{"Keep first error", newValidator().HasNamespace(&resource.Info{Namespace: "foo"}, "bar").HasName(&resource.Info{Name: "foo"}, "foo"), true},
	{"HaveSeen valid", newValidator().MarkSeen(namespaceType, "").HaveSeen(namespaceType), false},
	{"HaveSeen invalid", newValidator().HaveSeen(namespaceType), true},
	{"HaveNotSeen valid", newValidator().HaveNotSeen(namespaceType), false},
	{"HaveNotSeen invalid", newValidator().MarkSeen(namespaceType, "").HaveNotSeen(namespaceType), true},
	{"HaveNotSeenName valid", newValidator().MarkSeen(namespaceType, "foo").HaveNotSeenName(namespaceType, "bar"), false},
	{"HaveNotSeenName invalid", newValidator().MarkSeen(namespaceType, "foo").HaveNotSeenName(namespaceType, "foo"), true},
	{"ObjectDisallowedInContext", newValidator().ObjectDisallowedInContext(&resource.Info{Source: "some/source"}, namespaceType), true},
}

func TestValidator(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if !tc.expectedError && tc.v.err != nil {
				t.Fatalf("Expected error: %v", tc.v.err)
			}
			if tc.expectedError && tc.v.err == nil {
				t.Fatalf("Unexpected error")
			}
		})
	}
}
