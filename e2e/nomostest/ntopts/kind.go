package ntopts

import "testing"

// KindVersion is a specific Kind version associated with a Kubernetes minor version.
type KindVersion string

// The most recent
const (
	Kind1_18 KindVersion = "kindest/node:v1.18.2@sha256:7b27a6d0f2517ff88ba444025beae41491b016bc6af573ba467b70c5e8e0d85f"
	Kind1_17 KindVersion = "kindest/node:v1.17.5@sha256:ab3f9e6ec5ad8840eeb1f76c89bb7948c77bbf76bcebe1a8b59790b8ae9a283a"
	Kind1_16 KindVersion = "kindest/node:v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765"
	Kind1_15 KindVersion = "kindest/node:v1.15.11@sha256:6cc31f3533deb138792db2c7d1ffc36f7456a06f1db5556ad3b6927641016f50"
	Kind1_14 KindVersion = "kindest/node:v1.14.10@sha256:6cd43ff41ae9f02bb46c8f455d5323819aec858b99534a290517ebc181b443c6"
	Kind1_13 KindVersion = "kindest/node:v1.13.12@sha256:214476f1514e47fe3f6f54d0f9e24cfb1e4cda449529791286c7161b7f9c08e7"
	Kind1_12 KindVersion = "kindest/node:v1.12.10@sha256:faeb82453af2f9373447bb63f50bae02b8020968e0889c7fa308e19b348916cb"
)

// AsKindVersion returns the latest Kind version associated with a given
// Kubernetes minor version.
func AsKindVersion(t *testing.T, version string) KindVersion {
	t.Helper()

	switch version {
	case "1.12":
		return Kind1_12
	case "1.13":
		return Kind1_13
	case "1.14":
		return Kind1_14
	case "1.15":
		return Kind1_15
	case "1.16":
		return Kind1_16
	case "1.17":
		return Kind1_17
	case "1.18":
		return Kind1_18
	}
	t.Fatalf("Unrecognized Kind version: %q", version)
	return ""
}

// KindCluster are the options for setting up a Kind cluster.
type KindCluster struct {
	Version KindVersion
}
