package diff

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestIsUnknown(t *testing.T) {
	testCases := []struct {
		name string
		obj  core.Object
		want bool
	}{
		{
			"unknown object",
			Unknown(),
			true,
		},
		{
			"known object",
			fake.ConfigMapObject(),
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsUnknown(tc.obj)
			if got != tc.want {
				t.Errorf("got %v from IsUnknown(); want %v", got, tc.want)
			}
		})
	}
}
