package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestAddAnnotationsAndLabels(t *testing.T) {
	testcases := []struct {
		name       string
		actual     []core.Object
		expected   []core.Object
		gitRef     string
		gitRepo    string
		commitHash string
	}{
		{
			name:     "empty list",
			actual:   []core.Object{},
			expected: []core.Object{},
		},
		{
			name:       "nil annotation without env",
			gitRef:     "refs/head/master",
			gitRepo:    "git@github.com/foo",
			commitHash: "1234567",
			actual:     []core.Object{fake.RoleObject()},
			expected: []core.Object{fake.RoleObject(
				core.Label(v1.ManagedByKey, v1.ManagedByValue),
				core.Annotation(v1.ResourceManagementKey, "enabled"),
				core.Annotation(v1.ResourceManagerKey, "some-namespace"),
				core.Annotation(v1.SyncTokenAnnotationKey, "1234567"),
				core.Annotation(v1.GitRefKey, "refs/head/master"),
				core.Annotation(v1.GitRepoKey, "git@github.com/foo"),
			)},
		},
	}

	for _, tc := range testcases {
		addAnnotationsAndLabels(tc.actual, "some-namespace", tc.gitRef, tc.gitRepo, tc.commitHash)
		if diff := cmp.Diff(tc.expected, tc.actual); diff != "" {
			t.Errorf(diff)
		}
	}
}
