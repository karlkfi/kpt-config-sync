package k8s_test

import (
	"testing"

	"github.com/google/nomos/pkg/bespin-controllers/test/k8s"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsDeleted(t *testing.T) {
	nowTime := v1.Now()
	testCases := []struct {
		Name           string
		Time           *v1.Time
		ExpectedResult bool
	}{
		{"Nil time", nil, false},
		{"Now time", &nowTime, true},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			meta := v1.ObjectMeta{
				DeletionTimestamp: tc.Time,
			}
			result := k8s.IsDeleted(&meta)
			if result != tc.ExpectedResult {
				t.Errorf("result mismatch: got '%v', want '%v'", result, tc.ExpectedResult)
			}
		})
	}
}
