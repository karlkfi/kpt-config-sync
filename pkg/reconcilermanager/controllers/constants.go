package controllers

const (
	// GCPSAAnnotationKey is used to annotate RepoSync/RootSync controller SA when
	// spec.git.auth: gcpserviceaccount is used with Workload Identity enabled on a
	// GKE cluster.
	// https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
	GCPSAAnnotationKey = "iam.gke.io/gcp-service-account"
)

// Git secret configmap key names
const (
	// GitSecretConfigKeySSH is the key at which an ssh cert is stored
	GitSecretConfigKeySSH = "ssh"
	// GitSecretConfigKeyCookieFile is the key at which the git cookiefile is stored
	GitSecretConfigKeyCookieFile = "cookie_file"
	// GitSecretConfigKeyToken is the key at which a token's value is stored
	GitSecretConfigKeyToken = "token"
	// GitSecretConfigKeyTokenUsername is the key at which a token's username is stored
	GitSecretConfigKeyTokenUsername = "username"
)
