package webhook

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/testing/fake"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestValidatorForResourceGroups(t *testing.T) {
	testcases := []struct {
		name     string
		object   *unstructured.Unstructured
		groups   []string
		username string
		allow    bool
	}{
		{
			name: "request from ConfigSync: ResourceGroup generated by ConfigSync is allowed",
			object: fake.ResourceGroupObject(core.Name("repo-sync"), core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label(common.InventoryLabel, kptapplier.InventoryID("bookstore"))),
			groups: []string{saGroup, saNamespaceGroup},
			// TODO(jingfangliu): Use the ServiceAccount for repo reconciler after b/160786209 is resolved.
			username: saNamespaceGroup + ":importer",
			allow:    true,
		},
		{
			name: "request from ConfigSync: ResourceGroup not generated by ConfigSync is allowed",
			object: fake.ResourceGroupObject(core.Name("user-created"), core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			groups:   []string{saGroup, saNamespaceGroup},
			username: saNamespaceGroup + ":importer",
			allow:    true,
		},
		{
			name: "request not from ConfigSync: ResourceGroup generated by ConfigSync is denied",
			object: fake.ResourceGroupObject(core.Name("repo-sync"), core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
				core.Label(common.InventoryLabel, kptapplier.InventoryID("bookstore"))),
			username: "alice",
			allow:    false,
		},
		{
			name: "request not from ConfigSync: ResourceGroup not generated by ConfigSync is allowed",
			object: fake.ResourceGroupObject(core.Name("user-created"), core.Namespace("bookstore"),
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled)),
			username: "bob",
			allow:    true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			request := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   live.ResourceGroupGVK.Group,
						Version: live.ResourceGroupGVK.Version,
						Kind:    live.ResourceGroupGVK.Kind,
					},
					Name:      tc.object.GetName(),
					Namespace: tc.object.GetNamespace(),
					UserInfo: authenticationv1.UserInfo{
						Groups:   tc.groups,
						Username: tc.username,
					},
					Object: runtime.RawExtension{
						Object: tc.object,
					},
					OldObject: runtime.RawExtension{
						Object: tc.object,
					},
				},
			}
			v := &validator{}
			response := v.Handle(context.TODO(), request)
			if tc.allow != response.Allowed {
				t.Errorf("expected %v but got %v", tc.allow, response.Allowed)
			}
		})
	}
}
