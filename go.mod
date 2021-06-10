module github.com/google/nomos

go 1.15

// This should match the tag variable in generate-clientset.sh, and many of the
// k8s.io library versions below.
replace k8s.io/code-generator => k8s.io/code-generator v0.20.2

// Prevent Go from updating gnostic to a version incompatible with kube-openapi.
replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.1

// Keep the version of cli-runtime to v0.21.0 while keep other k8s dependency
// versions to v0.20.4
replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.0

replace k8s.io/api => k8s.io/api v0.20.4

replace k8s.io/apimachinery => k8s.io/apimachinery v0.20.4

replace k8s.io/client-go => k8s.io/client-go v0.20.4

require (
	cloud.google.com/go v0.72.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	github.com/GoogleContainerTools/kpt v0.37.1-0.20210128185716-8a1032f5571e
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/glogr v0.3.0
	github.com/go-logr/logr v0.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4
	github.com/googleapis/gnostic v0.5.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/open-policy-agent/cert-controller v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/cobra v1.1.1
	go.opencensus.io v0.22.5
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/oauth2 v0.0.0-20201109201403-9fd604954f58
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/api v0.36.0 // indirect
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f // indirect
	google.golang.org/grpc v1.35.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.21.0
	k8s.io/cli-runtime v0.20.4
	k8s.io/client-go v0.21.0
	k8s.io/cluster-registry v0.0.6
	k8s.io/kube-openapi v0.0.0-20210527164424-3c818078ee3d
	k8s.io/kubectl v0.20.4
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/cli-utils v0.24.0
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/kind v0.10.0
	sigs.k8s.io/structured-merge-diff/v4 v4.0.2
	sigs.k8s.io/yaml v1.2.0
)
