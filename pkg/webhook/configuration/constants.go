package configuration

import "github.com/google/nomos/pkg/api/configsync"

// ShortName is the short name of the ValidatingWebhookConfiguration for the
// Admission Controller.
const ShortName = "admission-webhook"

// Name is both:
// 1) The metadata.name of the ValidatingWebhookConfiguration, and
// 2) The .name of every ValidatingWebhook in the ValidatingWebhookConfiguration.
const Name = ShortName + "." + configsync.GroupName

// ServingPath is the path the webhook is served.
const ServingPath = "/" + ShortName

// Port matches the containerPort specified in admission-webhook.yaml.
const Port = 8676

// CertDir matches the mountPath specified in admission-webhook.yaml.
const CertDir = "/certs"

// DeclaredVersionLabel declares the API Version in which a resource was initially
// declared.
const DeclaredVersionLabel = configsync.GroupName + "/declared-version"

// WebhookconfigurationKey annotation declares if the webhook configuration
// should be updated.
const WebhookconfigurationKey = configsync.GroupName + "/webhook-configuration-update"

// WebhookConfigurationUpdateDisabled is the value for WebhookConfigurationKey
// to disable updating the webhook configuration.
const WebhookConfigurationUpdateDisabled = "disabled"
