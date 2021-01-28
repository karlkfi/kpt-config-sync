package webhook

import (
	"sort"

	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/status"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ToWebhookConfiguration creates a ValidatingWebhookConfiguration for all
// declared GroupVersionKinds.
//
// There is one ValidatingWebhook for every unique GroupVersion.
// Each ValidatingWebhook contains exactly one Rule with all Resources which
// correspond to the passed GroupVersionKinds.
func ToWebhookConfiguration(mapper kindResourceMapper, gvks []schema.GroupVersionKind) (admissionv1.ValidatingWebhookConfiguration, status.MultiError) {
	if len(gvks) == 0 {
		return admissionv1.ValidatingWebhookConfiguration{}, nil
	}

	webhookCfg := admissionv1.ValidatingWebhookConfiguration{}
	webhookCfg.SetNamespace(configsync.ControllerNamespace)
	webhookCfg.SetName(Name)

	webhooks, errs := newWebhooks(mapper, gvks)
	if errs != nil {
		return admissionv1.ValidatingWebhookConfiguration{}, errs
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
	return admissionv1.ValidatingWebhook{
		Name:  Name,
		Rules: []admissionv1.RuleWithOperations{toRule(gv)},
		// FailurePolicy is unset, so it defaults to Fail.
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"configmanagement.gke.io/declared-version": gv.Version,
			},
		},
		MatchPolicy: &equivalent,
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
