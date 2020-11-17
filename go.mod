module github.com/google/nomos

go 1.14

// The important one here is controller-runtime, which is pinned to the version
// in google3. Currently, this is v0.4.0. When updated (to v0.5.2), all of the
// other version tags will need to be updated to ones that work with it.
//
// https://cs.corp.google.com/piper///depot/google3/third_party/golang/kubecontrollerruntime/METADATA?rcl=304500938&l=15
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191004103531-b568748c9b85
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918162238-f783a3654da8
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269
)

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/glogr v0.1.0
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.4.0
	github.com/googleapis/gnostic v0.3.1
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/spf13/cobra v1.0.0
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073 // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.2
	k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783
	k8s.io/apimachinery v0.18.2
	k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828
	k8s.io/client-go v0.18.2
	k8s.io/cluster-registry v0.0.6
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c
	k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/kind v0.8.1
	sigs.k8s.io/yaml v1.2.0
)
