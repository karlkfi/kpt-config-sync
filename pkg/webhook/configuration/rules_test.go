package configuration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/kinds"
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
		name string
		gvks []schema.GroupVersionKind
		want *admissionv1.ValidatingWebhookConfiguration
	}{
		{
			name: "empty",
		},
		{
			name: "one GVK",
			gvks: []schema.GroupVersionKind{kinds.Role()},
			want: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Name,
					Namespace: configsync.ControllerNamespace,
				},
				Webhooks: []admissionv1.ValidatingWebhook{{
					Name:           webhookName(rbacv1.SchemeGroupVersion),
					MatchPolicy:    &equivalent,
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
				}},
			},
		},
		{
			name: "two GVKs same GV",
			gvks: []schema.GroupVersionKind{
				kinds.Role(),
				kinds.RoleBinding(),
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
							VersionLabel: "v1",
						},
					},
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
				}},
			},
		},
		{
			name: "two GVKs same Group",
			gvks: []schema.GroupVersionKind{
				kinds.Role(),
				kinds.RoleBindingV1Beta1(),
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      Name,
					Namespace: configsync.ControllerNamespace,
				},
				Webhooks: []admissionv1.ValidatingWebhook{{
					Name:           webhookName(rbacv1.SchemeGroupVersion),
					MatchPolicy:    &equivalent,
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
				}, {
					Name:           webhookName(rbacv1beta1.SchemeGroupVersion),
					MatchPolicy:    &equivalent,
					ObjectSelector: selectorFor(rbacv1beta1.SchemeGroupVersion.Version),
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1beta1.SchemeGroupVersion),
					},
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
							VersionLabel: "v1",
						},
					},
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
				}, {
					Name:        webhookName(corev1.SchemeGroupVersion),
					MatchPolicy: &equivalent,
					ObjectSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							VersionLabel: "v1",
						},
					},
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
					},
				},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := toWebhookConfiguration(tc.gvks)

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty(),
				cmpopts.IgnoreFields(admissionv1.ValidatingWebhook{}, "SideEffects", "AdmissionReviewVersions", "TimeoutSeconds"),
				cmpopts.IgnoreFields(admissionv1.WebhookClientConfig{}, "Service")); diff != "" {
				t.Errorf("TestToWebhookConfiguration() diff (-want +got):\n%s", diff)
			}
		})
	}
}
