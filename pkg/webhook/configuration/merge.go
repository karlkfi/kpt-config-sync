package configuration

import (
	"sort"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Merge merges two sets of ValidatingWebhookConfigurations so that the
// resulting Configuration matches the union of the initial two.
//
// The intent is that left is the Configuration currently on the server and
// right is the Configuration generated from the declared configuration in a
// repository.
// (The logic should be symmetric, so this shouldn't have to be the case.)
//
// The resulting merged Configuration meets the following criteria:
// 1) All Webhooks contain exactly one rule, matching a single GroupVersion of
//    declared configuration.
// 2) Webhooks are sorted by the GroupVersion they match.
// 3) Resources within Rules are sorted alphabetically.
//
// Cannot return error or panic as we never want this to get stuck.
//
// Modifies left.
func Merge(left, right *admissionv1.ValidatingWebhookConfiguration) *admissionv1.ValidatingWebhookConfiguration {
	webhooksMap := make(map[schema.GroupVersion]admissionv1.ValidatingWebhook)
	for _, webhook := range append(left.Webhooks, right.Webhooks...) {
		if len(webhook.Rules) == 0 ||
			len(webhook.Rules[0].APIGroups) == 0 ||
			len(webhook.Rules[0].APIVersions) == 0 {
			// Invalid ValidatingWebhook, discard.
			// We do NOT want to fail here as the Configuration on the API Server
			// may have been changed by a user. We don't want to put ourselves in an
			// infinite error loop.
			continue
		}

		groupVersion := gv(webhook)
		if oldWebhook, found := webhooksMap[groupVersion]; found {
			webhooksMap[groupVersion] = mergeWebhooks(oldWebhook, webhook)
		} else {
			webhooksMap[groupVersion] = webhook
		}
	}

	var webhooks []admissionv1.ValidatingWebhook
	for _, webhook := range webhooksMap {
		webhooks = append(webhooks, webhook)
	}
	sort.Slice(webhooks, func(i, j int) bool {
		groupI := webhooks[i].Rules[0].APIGroups[0]
		groupJ := webhooks[j].Rules[0].APIGroups[0]
		if groupI != groupJ {
			return groupI < groupJ
		}
		versionI := webhooks[i].Rules[0].APIVersions[0]
		versionJ := webhooks[j].Rules[0].APIVersions[0]
		return versionI < versionJ
	})

	left.Webhooks = webhooks
	return left
}

func mergeWebhooks(left, right admissionv1.ValidatingWebhook) admissionv1.ValidatingWebhook {
	switch {
	case len(left.Rules) == 0:
		left.Rules = right.Rules
	case len(right.Rules) == 0:
		// Nothing to merge.
	default:
		left.Rules[0] = mergeRule(left.Rules[0], right.Rules[0])
	}
	// Each Webhook should have exactly one Rule. Otherwise indicates a user
	// has modified the Webhook in a way we don't support.
	left.Rules = left.Rules[:1]

	// FailurePolicy Ignore wins, if set. Not strictly necessary, but here to make
	// the operation symmetric.
	if right.FailurePolicy != nil && *right.FailurePolicy == admissionv1.Ignore {
		ignore := admissionv1.Ignore
		left.FailurePolicy = &ignore
	}
	return left
}

// mergeRule merges the two RuleWithOperations together, modifying left.
// Returns the modified left.
func mergeRule(left, right admissionv1.RuleWithOperations) admissionv1.RuleWithOperations {
	// It would be more efficient to use a list merge algorithm as we know these
	// lists are sorted, but it is more difficult to read/maintain and the
	// performance gain is negligible.
	resourceSet := make(map[string]bool)
	var resources []string
	for _, resource := range append(left.Resources, right.Resources...) {
		if resourceSet[resource] {
			continue
		}
		resourceSet[resource] = true
		resources = append(resources, resource)
	}
	// Sort resources within the Rule.
	sort.Strings(resources)

	left.Resources = resources
	return left
}

// gv gets the GroupVersion a Webhook is for.
func gv(webhook admissionv1.ValidatingWebhook) schema.GroupVersion {
	// To be valid, Rules must contain at least one element in APIGroups and
	// APIVersions.
	return schema.GroupVersion{
		Group:   webhook.Rules[0].APIGroups[0],
		Version: webhook.Rules[0].APIVersions[0],
	}
}
