package configuration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/configsync"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
)

// toWebhookConfiguration creates a ValidatingWebhookConfiguration for all
// declared GroupVersionKinds.
//
// There is one ValidatingWebhook for every unique GroupVersion.
// Each ValidatingWebhook contains exactly one Rule with all Resources which
// correspond to the passed GroupVersionKinds.
func toWebhookConfiguration(gvks []schema.GroupVersionKind) *admissionv1.ValidatingWebhookConfiguration {
	if len(gvks) == 0 {
		return nil
	}

	webhookCfg := &admissionv1.ValidatingWebhookConfiguration{}
	webhookCfg.SetName(Name)
	webhookCfg.Webhooks = newWebhooks(gvks)

	return webhookCfg
}

// toRules converts a slice of GVRs to the corresponding admission Webhook
// Rules.
func newWebhooks(gvks []schema.GroupVersionKind) []admissionv1.ValidatingWebhook {
	if len(gvks) == 0 {
		return nil
	}

	seen := make(map[schema.GroupVersion]bool)
	var gvs []schema.GroupVersion
	for _, gvk := range gvks {
		gv := gvk.GroupVersion()
		if !seen[gv] {
			gvs = append(gvs, gv)
			seen[gv] = true
		}
	}

	// Sort GVRs lexicographically by Group/Version so the resulting list of
	// webhooks is deterministic.
	sort.Slice(gvs, func(i, j int) bool {
		return gvs[i].String() < gvs[j].String()
	})

	// Group Rules by GroupVersion. Each Rule corresponds to a single
	// Group/Version.
	var webhooks []admissionv1.ValidatingWebhook
	for _, gv := range gvs {
		webhooks = append(webhooks, toWebhook(gv))
	}
	return webhooks
}

// toWebhook creates a Webhook to match resources in the passed GroupVersion.
func toWebhook(gv schema.GroupVersion) admissionv1.ValidatingWebhook {
	// You cannot take address of constants in Go.
	equivalent := admissionv1.Equivalent
	none := admissionv1.SideEffectClassNone
	return admissionv1.ValidatingWebhook{
		Name:  webhookName(gv),
		Rules: []admissionv1.RuleWithOperations{ruleFor(gv)},
		// FailurePolicy is unset, so it defaults to Fail.
		ObjectSelector: selectorFor(gv.Version),
		// Match objects with the same GKNN but a different Version.
		MatchPolicy: &equivalent,
		// Checking to see if the update includes a conflict causes no side effects.
		SideEffects: &none,
		// Prefer v1 AdmissionReviews if available, otherwise fall back to v1beta1.
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
		ClientConfig: admissionv1.WebhookClientConfig{
			CABundle: []byte{},
			Service: &admissionv1.ServiceReference{
				Namespace: configsync.ControllerNamespace,
				Name:      ShortName,
				Path:      pointer.StringPtr(ServingPath),
				Port:      pointer.Int32Ptr(Port),
			},
		},
		// Several essential k8s checks require creating sequences of objects and
		// time out after 10 seconds, so 3 seconds is a reasonable upper bound.
		TimeoutSeconds: pointer.Int32Ptr(3),
	}
}

func ruleFor(gv schema.GroupVersion) admissionv1.RuleWithOperations {
	return admissionv1.RuleWithOperations{
		Rule: admissionv1.Rule{
			APIGroups:   []string{gv.Group},
			APIVersions: []string{gv.Version},
			Resources:   []string{"*"},
		},
		Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update, admissionv1.Delete},
	}
}

func selectorFor(version string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			DeclaredVersionLabel: version,
		},
	}
}

func webhookName(gv schema.GroupVersion) string {
	// Each Webhook in the WebhookConfiguration needs a unqiue name. We have
	// exactly one Webhook for each GroupVersion, so including both in the name
	// guarantees name uniqueness.
	if gv.Group != "" {
		return fmt.Sprintf("%s.%s.%s", strings.ToLower(gv.Group), strings.ToLower(gv.Version), Name)
	}
	// We can't start a Webhook name with a leading "."
	return fmt.Sprintf("%s.%s", strings.ToLower(gv.Version), Name)
}
