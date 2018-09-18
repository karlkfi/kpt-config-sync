package namespaces

import (
	"reflect"
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/cli"
	"github.com/google/nomos/pkg/client/meta/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type namespaceConfig struct {
	name   string
	parent string
}

func setUpNamespaces(t *testing.T, client clientv1.NamespaceInterface) {

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
		_, err := client.Create(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns.name,
				Labels: map[string]string{v1.ParentLabelKey: ns.parent}}})
		if err != nil {
			t.Errorf("Failed to create initial state: %v", err)
		}
	}

	_, err := client.Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "default"}})
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

// runNamespaceActions is the common part of the tests below.  testFunction is
// an adaptor that allows us to call different functions across test cases.
func runNamespaceActions(
	t *testing.T,
	testFunction func(
		fakeClient *fake.Client,
		namespaceInterface clientv1.NamespaceInterface,
		namespace string) ([]*corev1.Namespace, bool, error),
) {
	fakeClient := fake.NewClient()
	client := fakeClient.Kubernetes().CoreV1().Namespaces()

	setUpNamespaces(t, client)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {

			namespaces, includesRoot, err := testFunction(fakeClient, client, tc.namespace)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected function to return an error")
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

func TestNamespaceActions(t *testing.T) {
	runNamespaceActions(t, func(
		fakeClient *fake.Client,
		namespaceInterface clientv1.NamespaceInterface,
		namespace string) ([]*corev1.Namespace, bool, error) {
		return GetAncestry(namespaceInterface, namespace)
	})
}

func TestNamespaceActionsFromClient(t *testing.T) {
	runNamespaceActions(t, func(
		fakeClient *fake.Client,
		namespaceInterface clientv1.NamespaceInterface,
		namespace string) ([]*corev1.Namespace, bool, error) {
		ctx := cli.CommandContext{
			Client:    fakeClient,
			Namespace: namespace,
		}
		return GetAncestryFromContext(&ctx)
	})
}
