package configuration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/status"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// toWebhookConfiguration creates a ValidatingWebhookConfiguration for all
// declared GroupVersionKinds.
//
// There is one ValidatingWebhook for every unique GroupVersion.
// Each ValidatingWebhook contains exactly one Rule with all Resources which
// correspond to the passed GroupVersionKinds.
func toWebhookConfiguration(mapper kindResourceMapper, gvks []schema.GroupVersionKind) (*admissionv1.ValidatingWebhookConfiguration, status.MultiError) {
	if len(gvks) == 0 {
		return nil, nil
	}

	webhookCfg := &admissionv1.ValidatingWebhookConfiguration{}
	webhookCfg.SetNamespace(configsync.ControllerNamespace)
	webhookCfg.SetName(Name)

	webhooks, errs := newWebhooks(mapper, gvks)
	if errs != nil {
		return nil, errs
	}
	webhookCfg.Webhooks = webhooks

	return webhookCfg, nil
}

// newRules generates the set of required webhook rules to match all declared
// GroupVersionKinds. Each Rule is for a single GroupVersion.
//
// Returns an error if the API Resource List is malformed or there is a declared
// GVK that is not on the API Server.
func newWebhooks(mapper kindResourceMapper, gvks []schema.GroupVersionKind) ([]admissionv1.ValidatingWebhook, status.MultiError) {
	gvrs, err := toGVRs(mapper, gvks)
	if err != nil {
		// Parsing wasn't done properly. It's better to bail early than try to
		// apply invalid changes to the webhook.
		return nil, err
	}

	return toWebhooks(gvrs), nil
}

// toRules converts a slice of GVRs to the corresponding admission Webhook
// Rules.
func toWebhooks(gvrs []schema.GroupVersionResource) []admissionv1.ValidatingWebhook {
	if len(gvrs) == 0 {
		return nil
	}

	// Sort GVRs lexicographically by Group/Version/Resource.
	// This guarantees elements with the same GroupVersion are contiguous.
	// Also guarantees the resulting list of Webhooks are sorted by GroupVersion.
	sort.Slice(gvrs, func(i, j int) bool {
		return gvrs[i].String() < gvrs[j].String()
	})

	// Group Rules by GroupVersion. Each Rule corresponds to a single
	// Group/Version.
	var idx int
	gv := gvrs[0].GroupVersion()
	webhooks := []admissionv1.ValidatingWebhook{toWebhook(gv)}
	for _, gvr := range gvrs {
		if gvr.GroupVersion() != gv {
			// We're at a new GroupVersion, so create a new ValidatingWebhook.
			idx++
			gv = gvr.GroupVersion()
			webhooks = append(webhooks, toWebhook(gv))
		}
		webhooks[idx].Rules[0].Resources = append(webhooks[idx].Rules[0].Resources, gvr.Resource)
	}
	return webhooks
}

// toWebhook creates an empty Webhook to match resources in the passed
// GroupVersion.
//
// Resources is empty; at least one Resource must be added for the
// ValidatingWebhook to be valid.
func toWebhook(gv schema.GroupVersion) admissionv1.ValidatingWebhook {
	// You cannot take address of constants in Go.
	equivalent := admissionv1.Equivalent
	none := admissionv1.SideEffectClassNone
	return admissionv1.ValidatingWebhook{
		Name:  webhookName(gv),
		Rules: []admissionv1.RuleWithOperations{toRule(gv)},
		// FailurePolicy is unset, so it defaults to Fail.
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"configmanagement.gke.io/declared-version": gv.Version,
			},
		},
		// Match objects with the same GKNN but a different Version.
		MatchPolicy: &equivalent,
		// Checking to see if the update includes a conflict causes no side effects.
		SideEffects: &none,
		// Prefer v1 AdmissionReviews if available, otherwise fall back to v1beta1.
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
		ClientConfig: admissionv1.WebhookClientConfig{
			Service: &admissionv1.ServiceReference{
				Namespace: configsync.ControllerNamespace,
				Name:      Name,
			},
		},
	}
}

func toRule(gv schema.GroupVersion) admissionv1.RuleWithOperations {
	return admissionv1.RuleWithOperations{
		Rule: admissionv1.Rule{
			APIGroups:   []string{gv.Group},
			APIVersions: []string{gv.Version},
		},
		Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update, admissionv1.Delete},
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
