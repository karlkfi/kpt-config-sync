package differ

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/policycontroller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsManageableSystem(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		reserved bool
	}{
		{metav1.NamespaceDefault, true},
		{"foo-bar", false},
		{"kube-foo", false},
		{metav1.NamespacePublic, true},
		{metav1.NamespaceSystem, true},
		{policycontroller.NamespaceSystem, true},
		{configmanagement.ControllerNamespace, false},
	} {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			reserved := IsManageableSystemNamespace(testcase.name)
			if reserved != testcase.reserved {
				t.Errorf("Expected %v got %v", testcase.reserved, reserved)
			}
		})
	}
}
