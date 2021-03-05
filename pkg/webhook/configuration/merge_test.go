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
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "duplicate webhooks",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "different versions",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1beta1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1beta1.SchemeGroupVersion.Version),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}, {
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1beta1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1beta1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "different groups",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(corev1.SchemeGroupVersion.Version),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(corev1.SchemeGroupVersion.Version),
				}, {
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "honor FailurePolicy set to Ignore",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
					FailurePolicy:  &ignore,
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
					FailurePolicy:  &ignore,
				}},
			},
		},
		// Invalid Webhooks
		{
			name: "webhook missing Rules",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "drop webhook missing APIGroup",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
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
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "drop webhook missing APIVersion",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
					},
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "drop extra rules",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(corev1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(corev1.SchemeGroupVersion.Version),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(corev1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(corev1.SchemeGroupVersion.Version),
				}},
			},
		},
		{
			name: "drop duplicate rules",
			left: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			right: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
				}},
			},
			want: &admissionv1.ValidatingWebhookConfiguration{
				Webhooks: []admissionv1.ValidatingWebhook{{
					Rules: []admissionv1.RuleWithOperations{
						ruleFor(rbacv1.SchemeGroupVersion),
					},
					ObjectSelector: selectorFor(rbacv1.SchemeGroupVersion.Version),
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
