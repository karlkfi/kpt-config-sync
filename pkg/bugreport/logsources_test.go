package bugreport

import (
	"context"
	"sort"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

// Readables is a slice of Readable types
type Readables []Readable

// Sorting implementation allows for easy comparison during testing
func (rs Readables) Len() int {
	return len(rs)
}

func (rs Readables) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

func (rs Readables) Less(i, j int) bool {
	return rs[i].Name < rs[j].Name
}

func TestConvertLogSourcesToReadables(t *testing.T) {
	tests := []struct {
		name       string
		logSources logSources
		expected   Readables
		numErrors  int
	}{
		{
			name:       "logSources is empty.",
			logSources: logSources{},
			expected:   Readables{},
			numErrors:  0,
		},
		{
			name: "Non-empty log sources, no errors.",
			logSources: logSources{
				&mockLogSource{
					returnError: false,
					name:        "source_a",
					readCloser:  &mockReadCloser{},
				},
				&mockLogSource{
					returnError: false,
					name:        "source_b",
					readCloser:  &mockReadCloser{},
				},
				&mockLogSource{
					returnError: false,
					name:        "source_c",
					readCloser:  &mockReadCloser{},
				},
			},
			expected: Readables{
				{
					ReadCloser: &mockReadCloser{},
					Name:       "source_a",
				},
				{
					ReadCloser: &mockReadCloser{},
					Name:       "source_b",
				},
				{
					ReadCloser: &mockReadCloser{},
					Name:       "source_c",
				},
			},
			numErrors: 0,
		},
		{
			name: "Some working RCs, some errors.",
			logSources: logSources{
				&mockLogSource{
					returnError: true,
					name:        "source_a",
					readCloser:  &mockReadCloser{},
				},
				&mockLogSource{
					returnError: true,
					name:        "source_b",
					readCloser:  &mockReadCloser{},
				},
				&mockLogSource{
					returnError: false,
					name:        "source_c",
					readCloser:  &mockReadCloser{},
				},
				&mockLogSource{
					returnError: false,
					name:        "source_d",
					readCloser:  &mockReadCloser{},
				},
			},
			expected: Readables{
				{
					ReadCloser: &mockReadCloser{},
					Name:       "source_c",
				},
				{
					ReadCloser: &mockReadCloser{},
					Name:       "source_d",
				},
			},
			numErrors: 2,
		},
	}

	for _, test := range tests {
		test := test

		client := fake.NewSimpleClientset()

		t.Run(test.name, func(t *testing.T) {
			output, errorList := test.logSources.convertLogSourcesToReadables(context.Background(), client)

			if len(errorList) != test.numErrors {
				t.Errorf("Expected %v errors but received %v.", test.numErrors, len(errorList))
			}

			sort.Sort(Readables(output))
			sort.Sort(test.expected)

			if len(output) != len(test.expected) {
				t.Errorf("Expected expected Readables and actual readbles to have same length.")
				return
			}

			for i, o := range output {
				exp := test.expected[i]
				if exp != o {
					t.Errorf("Expected readable %v differs from actual: %v", exp, o)
					return
				}
			}
		})
	}
}
