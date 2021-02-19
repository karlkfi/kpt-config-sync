package configuration

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
)

func TestMerge(t *testing.T) {
	// Go doesn't allow taking the address of constants.
	ignore := admissionv1.Ignore

	testCases := []struct {
		name        string
		left, right *admissionv1.ValidatingWebhookConfiguration
		want        *admissionv1.ValidatingWebhookConfiguration
	}{
		{
			name:  "empty",
			left:  &admissionv1.ValidatingWebhookConfiguration{},
			right: &admissionv1.ValidatingWebhookConfiguration{},
			want:  &admissionv1.ValidatingWebhookConfiguration{},
		},
		{
			name: "one vs zero webhooks",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "duplicate webhooks",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "different resources",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "rolebindings"),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "rolebindings", "roles"),
				}},
			},
		},
		{
			name: "different versions",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1beta1.SchemeGroupVersion, "roles"),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}, {
					Rules: rules(rbacv1beta1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "different groups",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(corev1.SchemeGroupVersion, "namespaces"),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(corev1.SchemeGroupVersion, "namespaces"),
				}, {
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "honor FailurePolicy set to Ignore",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules:         rules(rbacv1.SchemeGroupVersion, "roles"),
					FailurePolicy: &ignore,
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules:         rules(rbacv1.SchemeGroupVersion, "roles"),
					FailurePolicy: &ignore,
				}},
			},
		},
		// Invalid Webhooks
		{
			name: "webhook missing Rules",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "drop webhook missing APIGroup",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{{
						Rule: admissionv1.Rule{
							APIVersions: []string{"v1"},
							Resources:   []string{"namespaces"},
						},
					}},
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "drop webhook missing APIVersion",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{{
						Rule: admissionv1.Rule{
							APIGroups: []string{""},
							Resources: []string{"namespaces"},
						},
					}},
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "roles"),
				}},
			},
		},
		{
			name: "drop extra rules",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: append(rules(rbacv1.SchemeGroupVersion, "roles"),
						rules(rbacv1.SchemeGroupVersion, "rolebindings")...),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: append(rules(rbacv1.SchemeGroupVersion, "clusterroles"),
						rules(rbacv1.SchemeGroupVersion, "clusterrolebindings")...),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: rules(rbacv1.SchemeGroupVersion, "clusterroles", "roles"),
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Merge(tc.left, tc.right)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}

			got = Merge(tc.right, tc.left)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("not symmetric: %s", diff)
			}
		})
	}
}
