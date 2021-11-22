module github.com/google/nomos

go 1.16

// Prevent Go from updating gnostic to a version incompatible with kube-openapi.
replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.1

// This should match the tag variable in generate-clientset.sh, and many of the
// k8s.io library versions below.
replace k8s.io/code-generator => k8s.io/code-generator v0.22.2

// The cluster is not configured to pull from gke-internal, so point
// to the local copy
replace gke-internal.googlesource.com/GoogleCloudPlatform/kustomize-metric-wrapper.git => ./private_repo/gke-internal.googlesource.com/GoogleCloudPlatform/kustomize-metric-wrapper.git

require (
	cloud.google.com/go v0.81.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	github.com/GoogleContainerTools/kpt v1.0.0-beta.12
	github.com/Masterminds/semver v1.5.0
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/logr v0.4.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.3.0
	github.com/googleapis/gnostic v0.5.5
	github.com/open-policy-agent/cert-controller v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	gke-internal.googlesource.com/GoogleCloudPlatform/kustomize-metric-wrapper.git v0.0.0-20211115233911-764f1f57167e
	go.opencensus.io v0.23.0
	go.uber.org/multierr v1.6.0
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	k8s.io/api v0.22.3
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.3
	k8s.io/cli-runtime v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/cluster-registry v0.0.6
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.10.0
	k8s.io/kube-openapi v0.0.0-20211109043139-026bd182f079
	k8s.io/kubectl v0.22.2
	k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e
	sigs.k8s.io/cli-utils v0.27.0
	sigs.k8s.io/controller-runtime v0.10.1
	sigs.k8s.io/kind v0.11.1
	sigs.k8s.io/kustomize/kyaml v0.13.1-0.20211203194734-cd2c6a1ad117
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2
	sigs.k8s.io/yaml v1.2.0
)
