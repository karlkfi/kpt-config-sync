package webhook

import "github.com/google/nomos/pkg/api/configsync"

// ShortName is the short name of the ValidatingWebhookConfiguration for the
// Admission Controller.
const ShortName = "config-sync-admitter"

// Name is both:
// 1) The metadata.name of the ValidatingWebhookConfiguration, and
// 2) The .name of every ValidatingWebhook in the ValidatingWebhookConfiguration.
const Name = ShortName + "." + configsync.GroupName
