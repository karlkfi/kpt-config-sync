package namespaceconfig

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/testing/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespaceConfigEqualTestcase struct {
	name      string
	lhs       *v1.NamespaceConfig
	rhs       *v1.NamespaceConfig
	wantEqual bool
}

func (t *namespaceConfigEqualTestcase) Run(tt *testing.T) {
	equal, err := NamespaceConfigsEqual(fake.NewDecoder(nil), t.lhs, t.rhs)
	if err != nil {
		tt.Fatalf("unexpected error: %v", err)
	}

	if t.wantEqual == equal {
		return
	}

	diff := cmp.Diff(t.lhs, t.rhs, ncsIgnore...)
	if equal {
		tt.Errorf("wanted not equal, got equal: %s", diff)
	} else {
		tt.Errorf("wanted equal, got not equal: %s", diff)
	}
}

var time1 = time.Now()

var namespaceConfigEqualTestcases = []namespaceConfigEqualTestcase{
	{
		name: "basic",
		lhs: &v1.NamespaceConfig{
			Spec: v1.NamespaceConfigSpec{},
		},
		rhs: &v1.NamespaceConfig{
			Spec: v1.NamespaceConfigSpec{},
		},
		wantEqual: true,
	},
	{
		name: "different import token",
		lhs: &v1.NamespaceConfig{
			Spec: v1.NamespaceConfigSpec{
				Token: "1234567890123456789012345678901234567890",
			},
		},
		rhs: &v1.NamespaceConfig{
			Spec: v1.NamespaceConfigSpec{
				Token: "1234567890123456789012345678900000000000",
			},
		},
		wantEqual: true,
	},
	{
		name: "different import time",
		lhs: &v1.NamespaceConfig{
			Spec: v1.NamespaceConfigSpec{
				ImportTime: metav1.NewTime(time1),
			},
		},
		rhs: &v1.NamespaceConfig{
			Spec: v1.NamespaceConfigSpec{
				ImportTime: metav1.NewTime(time1.Add(time.Second)),
			},
		},
		wantEqual: true,
	},
}

func TestNamespaceConfigEqual(t *testing.T) {
	for _, tc := range namespaceConfigEqualTestcases {
		t.Run(tc.name, tc.Run)
	}
}
