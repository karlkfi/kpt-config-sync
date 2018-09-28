// Package config contains the types used to parse and embed a Google-compatible
// client ID configuration file.
//
// DO NOT MOVE THESE OUT OF PACKAGE DOCSTRING
// +k8s:deepcopy-gen=package,register
// +groupName=oidc.nomos.dev
package config

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

// To regenerate, run:
//    go get github.com/code-generator/cmd/deepcopy-gen
//    go install github.com/code-generator/cmd/deepcopy-gen
//    go generate github.com/google/nomos/pkg/oidc/config
//
// This was not made part of the automated generation, because this code is not
// expected to change often.
//
//go:generate deepcopy-gen --logtostderr -v=5 --input-dirs=. --output-file-base=types.generated --output-package="github.com/google/nomos/pkg/oidc/config"

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "oidc.nomos.dev", Version: "v1alpha1"}

	// SchemeBuilder is the scheme builder for types in this package
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme  adds the types in this package to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	// Makes it possible for the "clientcmd" scheme to decode our client ID
	// data as an extension
	if err := AddToScheme(clientcmdlatest.Scheme); err != nil {
		panic(err)
	}
}

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns  Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &ClientID{}, &InstalledClientSpec{})
	err := scheme.AddConversionFuncs(
		func(in *ClientID, out *runtime.RawExtension, _ conversion.Scope) error {
			out.Object = in
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("AddConversionFuncs: %v", err)
	}
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// ClientID is a partial client ID configuration.  The format of this type
// has been defined externally, and we are only declaring the parts of it that
// are relevant in the OIDC use case.
//
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClientID struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// +optional
	Installed *InstalledClientSpec `json:"installed"`
}

// InstalledClientSpec is the client ID configuration for a client ID that is
// of the "installed" type.
//
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type InstalledClientSpec struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// +optional
	ClientID string `json:"client_id"`

	// +optional
	ClientSecret string `json:"client_secret"`
}
