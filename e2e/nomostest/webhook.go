package nomostest

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// WaitForWebhookReadiness waits up to 3 minutes for the wehbook becomes ready.
// If the webhook still is not ready after 3 minutes, the test would fail.
func WaitForWebhookReadiness(nt *NT) {
	nt.T.Logf("Waiting for the webhook becomes ready: %v", time.Now())
	_, err := Retry(3*time.Minute, func() error {
		return webhookReadiness(nt)
	})
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.T.Logf("The webhook is ready at %v", time.Now())
}

func webhookReadiness(nt *NT) error {
	out, err := nt.Kubectl("logs", "-n", "config-management-system", "-l", "app=admission-webhook", "--tail=-1")
	if err != nil {
		nt.T.Fatalf("`kubectl logs -n config-management-system -l app=admission-webhook --tail=-1` failed: %v", err)
	}
	readyStr := `controller-runtime/webhook "level"=0 "msg"="serving webhook server"  "host"="" "port"=10250`
	if !strings.Contains(string(out), readyStr) {
		return errors.Errorf("The webhook is not ready yet")
	}
	return nil
}

// StopWebhook removes the Config Sync ValidatingWebhookConfiguration object.
func StopWebhook(nt *NT) {
	webhookName := configuration.Name
	webhookGK := "validatingwebhookconfigurations.admissionregistration.k8s.io"

	out, err := nt.Kubectl("annotate", webhookGK, webhookName, fmt.Sprintf("%s=%s", metadata.WebhookconfigurationKey, metadata.WebhookConfigurationUpdateDisabled))
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate %s %s %s=%s` error %v %s, want return nil",
			webhookGK, webhookName, metadata.WebhookconfigurationKey, metadata.WebhookConfigurationUpdateDisabled, err, out)
	}

	_, err = Retry(30*time.Second, func() error {
		return nt.Validate(webhookName, "", &admissionv1.ValidatingWebhookConfiguration{},
			HasAnnotation(metadata.WebhookconfigurationKey, metadata.WebhookConfigurationUpdateDisabled))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	out, err = nt.Kubectl("delete", webhookGK, webhookName)
	if err != nil {
		nt.T.Fatalf("got `kubectl delete %s %s` error %v %s, want return nil", webhookGK, webhookName, err, out)
	}

	_, err = Retry(30*time.Second, func() error {
		return nt.ValidateNotFound(webhookName, "", &admissionv1.ValidatingWebhookConfiguration{})
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func installWebhook(nt *NT, nomos ntopts.Nomos) {
	nt.T.Helper()
	objs := parseManifests(nt, nomos)
	for _, o := range objs {
		labels := o.GetLabels()
		if labels == nil || labels["app"] != "admission-webhook" {
			continue
		}
		nt.T.Logf("installWebhook obj: %v", core.GKNN(o))
		err := nt.Create(o)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}
			nt.T.Fatal(err)
		}
	}
}
