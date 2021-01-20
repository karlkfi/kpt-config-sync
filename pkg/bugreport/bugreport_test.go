package bugreport

import (
	"context"
	"fmt"
	"io"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/testing/fake"
	v1 "k8s.io/api/core/v1"
)

func TestAssembleLogSources(t *testing.T) {
	tests := []struct {
		name           string
		ns             v1.Namespace
		pods           v1.PodList
		expectedValues logSources
	}{
		{
			name:           "No pods",
			ns:             *fake.NamespaceObject("foo"),
			pods:           v1.PodList{Items: make([]v1.Pod, 0)},
			expectedValues: make(logSources, 0),
		},
		{
			name: "Multiple pods with various container configurations",
			ns:   *fake.NamespaceObject("foo"),
			pods: v1.PodList{Items: []v1.Pod{
				*fake.PodObject("foo_a", []v1.Container{
					*fake.ContainerObject("1"),
					*fake.ContainerObject("2"),
				}),
				*fake.PodObject("foo_b", []v1.Container{
					*fake.ContainerObject("3"),
				}),
			}},
			expectedValues: logSources{
				&logSource{
					ns: *fake.NamespaceObject("foo"),
					pod: *fake.PodObject("foo_a", []v1.Container{
						*fake.ContainerObject("1"),
						*fake.ContainerObject("2"),
					}),
					cont: *fake.ContainerObject("1"),
				},
				&logSource{
					ns: *fake.NamespaceObject("foo"),
					pod: *fake.PodObject("foo_a", []v1.Container{
						*fake.ContainerObject("1"),
						*fake.ContainerObject("2"),
					}),
					cont: *fake.ContainerObject("2"),
				},
				&logSource{
					ns: *fake.NamespaceObject("foo"),
					pod: *fake.PodObject("foo_b", []v1.Container{
						*fake.ContainerObject("3"),
					}),
					cont: *fake.ContainerObject("3"),
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			outputs := assembleLogSources(test.ns, test.pods)

			sort.Sort(outputs)
			sort.Sort(test.expectedValues)

			for i, output := range outputs {
				expected := test.expectedValues[i]
				if diff := cmp.Diff(output, expected, cmp.AllowUnexported(logSource{})); diff != "" {
					t.Errorf("%T differ (-got, +want): %s", expected, diff)
				}
			}
		})
	}
}

type mockLogSource struct {
	returnError bool
	name        string
	readCloser  io.ReadCloser
}

var _ convertibleLogSourceIdentifiers = &mockLogSource{}

// fetchRcForLogSource implements convertibleLogSourceIdentifiers.
func (m *mockLogSource) fetchRcForLogSource(ctx context.Context, cs coreClient) (io.ReadCloser, error) {
	if m.returnError {
		return nil, fmt.Errorf("failed to get RC")
	}

	return m.readCloser, nil
}

func (m *mockLogSource) pathName() string {
	return m.name
}

type mockReadCloser struct{}

var _ io.ReadCloser = &mockReadCloser{}

// Read implements io.ReadCloser.
func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, nil
}

// Close implements io.ReadCloser.
func (m *mockReadCloser) Close() error {
	return nil
}

// Sorting implementation allows for easy comparison during testing
func (ls logSources) Len() int {
	return len(ls)
}

func (ls logSources) Swap(i, j int) {
	ls[i], ls[j] = ls[j], ls[i]
}

func (ls logSources) Less(i, j int) bool {
	return ls[i].pathName() < ls[j].pathName()
}
