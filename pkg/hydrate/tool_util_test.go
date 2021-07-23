package hydrate

import (
	"errors"
	"testing"
)

func TestValidateTool(t *testing.T) {
	testCases := []struct {
		name             string
		toolVersion      string
		toolVersionError error
		requiredVersion  string
		expectedErr      error
	}{
		{
			name:             "tool not installed",
			toolVersionError: errors.New("command not found"),
			requiredVersion:  KustomizeVersion,
			expectedErr:      errors.New("command not found"),
		},
		{
			name:            "tool version is too old",
			toolVersion:     "v3.6.5",
			requiredVersion: KustomizeVersion,
			expectedErr:     errors.New(`the current kustomize version is "3.6.5". Please upgrade to version v4.1.3+`),
		},
		{
			name:            "tool version is the same as required",
			toolVersion:     KustomizeVersion,
			requiredVersion: KustomizeVersion,
		},
		{
			name:            "tool version is newer than required",
			toolVersion:     "v4.4.4",
			requiredVersion: KustomizeVersion,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toolVersion = func(string) (string, error) {
				return tc.toolVersion, tc.toolVersionError
			}
			err := ValidateTool(Kustomize, tc.requiredVersion)
			if err != nil && tc.expectedErr == nil {
				t.Errorf("%s: expected no error, but got error: %v", tc.name, err)
			} else if err == nil && tc.expectedErr != nil {
				t.Errorf("%s: got no error, but expected error: %v", tc.name, tc.expectedErr)
			} else if err != nil && tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() {
				t.Errorf("%s: got error: %v, but expected: %v", tc.name, err, tc.expectedErr)
			}
		})
	}
}

func TestNeedsKustomize(t *testing.T) {
	testCases := []struct {
		name   string
		dir    string
		result bool
	}{
		{
			name:   "A wet repo doesn't need kustomization",
			dir:    "../../e2e/testdata/hydration/wet-repo",
			result: false,
		},
		{
			name:   "A repo has a kustomization.yaml file",
			dir:    "../../e2e/testdata/hydration/helm-components",
			result: true,
		},
		{
			name:   "A repo has a kustomization.yml file",
			dir:    "../../e2e/testdata/hydration/kustomize-components",
			result: true,
		},
		{
			name:   "A repo has a kustomization.yml file in the nested directory",
			dir:    "../../e2e/testdata/hydration",
			result: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			needs, err := NeedsKustomize(tc.dir)
			if err != nil {
				t.Errorf("%s: expected no error, but got error: %v", tc.name, err)
			} else if needs != tc.result {
				t.Errorf("%s: expected %t, but got %t", tc.name, tc.result, needs)
			}
		})
	}
}
