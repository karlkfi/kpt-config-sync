package namespacewalker

import (
	"github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/meta/fake"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"reflect"
	"testing"
)

type namespaceConfig struct {
	name   string
	parent string
}

func setUpNamespaces(t *testing.T, client client_v1.NamespaceInterface) {

	for _, ns := range []namespaceConfig{
		{
			name:   "frontend",
			parent: "eng",
		},
		{
			name:   "eng",
			parent: "acme",
		},
		{
			name:   "acme",
			parent: "",
		},
	} {
		_, err := client.Create(&core_v1.Namespace{
			ObjectMeta: meta_v1.ObjectMeta{Name: ns.name,
				Labels: map[string]string{v1.ParentLabelKey: ns.parent}}})
		if err != nil {
			t.Errorf("Failed to create initial state: %v", err)
		}
	}

	_, err := client.Create(&core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{Name: "default"}})
	if err != nil {
		t.Errorf("Failed to create initial state: %v", err)
	}
}

type testCase struct {
	testName             string
	namespace            string
	expectedNamespaces   []string
	expectedIncludesRoot bool
	expectedError        bool
}

var testCases = []testCase{
	{
		"Leaf namespace",
		"frontend",
		[]string{"frontend", "eng", "acme"},
		true,
		false,
	},
	{
		"Intermediate namespace",
		"eng",
		[]string{"eng", "acme"},
		true,
		false,
	},
	{
		"Root namespace",
		"acme",
		[]string{"acme"},
		true,
		false,
	},
	{
		"Invalid namespace",
		"covfefe",
		nil,
		false,
		true,
	},
	{
		"Default namespace",
		"default",
		nil,
		false,
		true,
	},
}

func TestNamespaceActions(t *testing.T) {
	client := fake.NewClient().Kubernetes().CoreV1().Namespaces()

	setUpNamespaces(t, client)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			namespaces, includesRoot, err := GetAncestry(client, tc.namespace)
			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected fucntion to return an error")
				}
				return
			} else if !tc.expectedError && err != nil {
				t.Errorf("Unexpected error return by function")
				return
			}

			if includesRoot != tc.expectedIncludesRoot {
				t.Errorf("Unexpected includesRoot returned")
			}

			names := make([]string, 0)
			for _, ns := range namespaces {
				names = append(names, ns.Name)
			}

			if !reflect.DeepEqual(names, tc.expectedNamespaces) {
				t.Errorf("Returned namespaces is not correct")
			}
		})
	}
}
