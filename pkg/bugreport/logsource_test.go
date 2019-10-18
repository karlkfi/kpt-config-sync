package bugreport

import (
	"testing"

	"github.com/google/nomos/pkg/testing/fake"
	v1 "k8s.io/api/core/v1"
)

func TestLogSourceGetPathName(t *testing.T) {
	tests := []struct {
		name         string
		source       logSource
		expectedName string
	}{
		{
			name: "valid loggable produces hyphenated name",
			source: logSource{
				ns:   *fake.NamespaceObject("myNamespace"),
				pod:  *fake.PodObject("myPod", make([]v1.Container, 0)),
				cont: *fake.ContainerObject("myContainer"),
			},
			expectedName: "myNamespace/myPod/myContainer",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			actualName := test.source.pathName()
			if actualName != test.expectedName {
				t.Errorf("Expected loggable name to be %v, but received %v", test.expectedName, actualName)
			}
		})
	}
}
