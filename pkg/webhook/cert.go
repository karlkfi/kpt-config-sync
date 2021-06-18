package webhook

import (
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/webhook/configuration"
	cert "github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	secret         = configuration.ShortName + "-cert"
	caName         = "config-sync-ca"
	caOrganization = "config-sync"

	// dnsName is <service name>.<namespace>.svc
	dnsName = configuration.ShortName + "." + constants.ControllerNamespace + ".svc"
)

// CreateCertsIfNeeded creates all certs for webhooks.
// This function is called from main.go
func CreateCertsIfNeeded(mgr manager.Manager, restartOnSecretRefresh bool) (chan struct{}, error) {
	setupFinished := make(chan struct{})
	err := cert.AddRotator(mgr, &cert.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: constants.ControllerNamespace,
			Name:      secret,
		},
		CertDir:        configuration.CertDir,
		CAName:         caName,
		CAOrganization: caOrganization,
		DNSName:        dnsName,
		IsReady:        setupFinished,
		Webhooks: []cert.WebhookInfo{{
			Type: cert.Validating,
			Name: configuration.Name,
		}},
		RestartOnSecretRefresh: restartOnSecretRefresh,
	})
	return setupFinished, err
}
