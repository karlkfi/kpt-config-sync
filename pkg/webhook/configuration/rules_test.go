package configuration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestToWebhookConfiguration(t *testing.T) {
	equivalent := admissionv1.Equivalent

	testCases := []struct {
		name         string
		apiResources []*metav1.APIResourceList
		gvks         []schema.GroupVersionKind
		want         *admissionv1.ValidatingWebhookConfiguration
		wantErr      status.MultiError
	}{
		{
			name: "empty",
		},
		{
			name:    "one GVK missing API Resource",
			gvks:    []schema.GroupVersionKind{kinds.Role()},
			wantErr: status.InternalError(""),
		},
		{
			name: "one GVK",
			gvks: []schema.GroupVersionKind{kinds.Role()},
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion, apiResource("roles", "Role")),
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Name,
					Namespace: configsync.ControllerNamespace,
				},
				Webhooks: []admissionv1.ValidatingWebhook{{
					Name:        webhookName(rbacv1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"configmanagement.gke.io/declared-version": "v1",
						},
					},
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "two GVKs same GV",
			gvks: []schema.GroupVersionKind{
				kinds.Role(),
				kinds.RoleBinding(),
			},
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role"),
					apiResource("rolebindings", "RoleBinding")),
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Name,
					Namespace: configsync.ControllerNamespace,
				},
				Webhooks: []admissionv1.ValidatingWebhook{{
					Name:        webhookName(rbacv1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"configmanagement.gke.io/declared-version": "v1",
						},
					},
					Rules: rules(rbacv1.SchemeGroupVersion, "rolebindings", "roles"),
				}},
			},
		},
		{
			name: "two GVKs same Group",
			gvks: []schema.GroupVersionKind{
				kinds.Role(),
				kinds.RoleBindingV1Beta1(),
			},
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role")),
				apiResourceList(rbacv1beta1.SchemeGroupVersion,
					apiResource("rolebindings", "RoleBinding")),
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Name,
					Namespace: configsync.ControllerNamespace,
				},
				Webhooks: []admissionv1.ValidatingWebhook{{
					Name:        webhookName(rbacv1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"configmanagement.gke.io/declared-version": "v1",
						},
					},
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}, {
					Name:        webhookName(rbacv1beta1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"configmanagement.gke.io/declared-version": "v1beta1",
						},
					},
					Rules: rules(rbacv1beta1.SchemeGroupVersion, "rolebindings"),
				},
				},
			},
		},
		{
			name: "two GVKs same Version",
			gvks: []schema.GroupVersionKind{
				kinds.Role(),
				kinds.Namespace(),
			},
			apiResources: []*metav1.APIResourceList{
				apiResourceList(rbacv1.SchemeGroupVersion,
					apiResource("roles", "Role")),
				apiResourceList(corev1.SchemeGroupVersion,
					apiResource("namespaces", "Namespace")),
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Name,
					Namespace: configsync.ControllerNamespace,
				},
				Webhooks: []admissionv1.ValidatingWebhook{{
					Name:        webhookName(corev1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"configmanagement.gke.io/declared-version": "v1",
						},
					},
					Rules: rules(corev1.SchemeGroupVersion, "namespaces"),
				}, {
					Name:        webhookName(rbacv1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"configmanagement.gke.io/declared-version": "v1",
						},
					},
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapper, err := newKindResourceMapper(tc.apiResources)
			if err != nil {
				t.Fatalf("creating mapper: %v", err)
			}
			got, err := toWebhookConfiguration(mapper, tc.gvks)

			if diff := cmp.Diff(got, tc.want, cmpopts.EquateEmpty(),
				cmpopts.IgnoreFields(admissionv1.ValidatingWebhook{}, "SideEffects", "AdmissionReviewVersions", "TimeoutSeconds"),
				cmpopts.IgnoreFields(admissionv1.WebhookClientConfig{}, "Service")); diff != "" {
				t.Errorf("TestToWebhookConfiguration() diff (-want +got):\n%s", diff)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got TestToWebhookConfiguration() err %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func rules(gv schema.GroupVersion, resources ...string) []admissionv1.RuleWithOperations {
	// This means we aren't really testing the internals of toRule(), but it's
	// small enough to not worry about. We could manually specify this in every
	// test but it harms test readability.
	result := toRule(gv)
	result.Resources = resources
	return []admissionv1.RuleWithOperations{result}
}

func apiResource(name, kind string) metav1.APIResource {
	return metav1.APIResource{
		Name: name,
		Kind: kind,
	}
}

func apiResourceList(gv schema.GroupVersion, resources ...metav1.APIResource) *metav1.APIResourceList {
	return &metav1.APIResourceList{
		GroupVersion: gv.String(),
		APIResources: resources,
	}
}
