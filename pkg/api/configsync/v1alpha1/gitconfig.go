package v1alpha1

import (
	"github.com/google/nomos/pkg/api/configsync"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Git contains the configs which specify how to connect to and read from a Git
// repository.
type Git struct {
	// Repo is the git repository URL to sync from. Required.
	Repo string `json:"repo"`

	// Branch is the git branch to checkout. Default: "master".
	// +optional
	Branch string `json:"branch,omitempty"`

	// Revision is the git revision (tag, ref or commit) to fetch. Default: "HEAD".
	// +optional
	Revision string `json:"revision,omitempty"`

	// Dir is the absolute path of the directory that contains
	// the local policy.  Default: the root directory of the repo.
	// +optional
	Dir string `json:"dir,omitempty"`

	// Period is the time duration in seconds between consecutive
	// syncs.  Default: 15 seconds.
	// Note to developers that customers specify this value using
	// string (https://golang.org/pkg/time/#Duration.String) like "3s"
	// in their Custom Resource YAML. However, time.Duration is at a nanosecond
	// granularity, and it's easy to introduce a bug where it looks like the
	// code is dealing with seconds but its actually nanoseconds (or vice versa).
	// +optional
	Period metav1.Duration `json:"period,omitempty"`

	// Auth is the type of secret configured for access to the Git repo.
	// Must be one of ssh, cookiefile, gcenode, token, or none. Required.
	// The validation of this is case-sensitive. Required.
	//
	// +kubebuilder:validation:Pattern=^(ssh|cookiefile|gcenode|gcpserviceaccount|token|none)$
	Auth string `json:"auth"`

	// GCPServiceAccountEmail specifies the GCP service account used to annotate
	// the RootSync/RepoSync controller Kubernetes Service Account.
	// Note: The field is used when spec.git.auth: gcpserviceaccount.
	GCPServiceAccountEmail string `json:"gcpServiceAccountEmail,omitempty"`

	// Proxy is a struct that contains options for configuring access to the Git repo via a proxy.
	// Only has an effect when secretType is one of ("cookiefile", "none", "token"). Optional.
	// +optional
	Proxy string `json:"proxy,omitempty"`

	// SecretRef is the secret used to connect to the Git source of truth.
	// +optional
	SecretRef SecretReference `json:"secretRef,omitempty"`

	// noSSLVerify specifies whether to enable or disable the SSL certificate verification. Default: false.
	// If noSSLVerify is set to true, it tells Git to skip the SSL certificate verification.
	// +optional
	NoSSLVerify bool `json:"noSSLVerify,omitempty"`
}

// SecretReference contains the reference to the secret used to connect to
// Git source of truth.
type SecretReference struct {
	// Name represents the secret name.
	// +optional
	Name string `json:"name,omitempty"`
}

// GetPeriodSecs returns the sync period defaulting to 15 if empty.
func GetPeriodSecs(g *Git) float64 {
	if g.Period.Duration == 0 {
		return configsync.DefaultPeriodSecs
	}
	return g.Period.Duration.Seconds()
}
