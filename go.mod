module github.com/google/nomos

go 1.16

// Prevent Go from updating gnostic to a version incompatible with kube-openapi.
replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.1

// This should match the tag variable in generate-clientset.sh, and many of the
// k8s.io library versions below.
replace k8s.io/code-generator => k8s.io/code-generator v0.21.1

// The cluster is not configured to pull from gke-internal, so point
// to the local copy
replace gke-internal.googlesource.com/GoogleCloudPlatform/kustomize-metric-wrapper.git => ./private_repo/gke-internal.googlesource.com/GoogleCloudPlatform/kustomize-metric-wrapper.git

require (
	cloud.google.com/go v0.72.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	github.com/GoogleContainerTools/kpt v1.0.0-beta.7
	github.com/Masterminds/semver v1.5.0
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/glogr v0.3.0
	github.com/go-logr/logr v0.4.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/uuid v1.2.0
	github.com/googleapis/gnostic v0.5.5
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/open-policy-agent/cert-controller v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/cobra v1.1.3
	gke-internal.googlesource.com/GoogleCloudPlatform/kustomize-metric-wrapper.git v0.0.0-20211115233911-764f1f57167e
	go.opencensus.io v0.23.0
	go.uber.org/multierr v1.5.0
	golang.org/x/oauth2 v0.0.0-20201109201403-9fd604954f58
	google.golang.org/api v0.36.0 // indirect
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f // indirect
	google.golang.org/grpc v1.35.0 // indirect
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/cli-runtime v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/cluster-registry v0.0.6
	k8s.io/kube-openapi v0.0.0-20210527164424-3c818078ee3d
	k8s.io/kubectl v0.21.1
	k8s.io/utils v0.0.0-20210707171843-4b05e18ac7d9
	sigs.k8s.io/cli-utils v0.26.1
	sigs.k8s.io/controller-runtime v0.9.0-beta.5.0.20210524185538-7181f1162e79
	sigs.k8s.io/kind v0.11.1
	sigs.k8s.io/kustomize/kyaml v0.11.2-0.20210920224623-c47fc4860720
	sigs.k8s.io/structured-merge-diff/v4 v4.1.0
	sigs.k8s.io/yaml v1.2.0
)
