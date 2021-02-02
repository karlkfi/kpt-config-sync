module github.com/google/nomos

go 1.15

// This should match the tag variable in generate-clientset.sh, and many of the
// k8s.io library versions below.
replace k8s.io/code-generator => k8s.io/code-generator v0.19.4

// This is to fix the failure in `make build-cli` that is related to
// github.com/moby/term (b/177232366).
replace golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6

require (
	contrib.go.opencensus.io/exporter/ocagent v0.7.0
	github.com/GoogleContainerTools/kpt v0.37.1-0.20210128185716-8a1032f5571e
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/glogr v0.3.0
	github.com/go-logr/logr v0.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.5.4
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/cobra v1.1.1
	go.opencensus.io v0.22.5
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210113181707-4bcb84eeeb78 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/api v0.36.0 // indirect
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f // indirect
	google.golang.org/grpc v1.35.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/cli-runtime v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/cluster-registry v0.0.6
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd
	k8s.io/kubectl v0.19.4
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	sigs.k8s.io/cli-utils v0.22.4-0.20210108175429-beb6f88a4384
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/kind v0.10.0
	sigs.k8s.io/yaml v1.2.0
)
