package kptfile

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Group is the kpt API GRoup.
	Group = "kpt.dev"
	// Kind is the Kptfile Kind string.
	Kind = "Kptfile"
)

// SchemeGroupVersion is the kpt GroupVersion we're compatible with.
var SchemeGroupVersion = schema.GroupVersion{
	Group:   Group,
	Version: "v1alpha1",
}

// PackageMeta defines the metadata of a package.
type PackageMeta struct {
	ShortDescription string `json:"shortDescription,omitempty"`
}

// Inventory contains the information to generate a ResourceGroup.
type Inventory struct {
	Identifier  string            `json:"identifier"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Kptfile defines a Kptfile that is read from a github repo.
// The type is from https://googlecontainertools.github.io/kpt/api-reference/kptfile/.
// Here is an example
//    apiVersion: kpt.dev/v1alpha1
//    kind: Kptfile
//    metadata:
//      name: package-name
//    packageMetadata:
//      shortDescription: This is a description
//    inventory:
//      identifier: some-name
//      namespace: foo
//      labels:
//        sonic: youth
//      annotations:
//        husker: du
type Kptfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	PackageMetadata   PackageMeta `json:"packageMetadata,omitempty"`
	Inventory         Inventory   `json:"inventory,omitempty"`
}

func (in *Kptfile) deepCopy() *Kptfile {
	if in == nil {
		return nil
	}
	out := new(Kptfile)
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.PackageMetadata = PackageMeta{
		ShortDescription: in.PackageMetadata.ShortDescription,
	}
	out.Inventory = Inventory{
		Identifier:  in.Inventory.Identifier,
		Namespace:   in.Inventory.Namespace,
		Labels:      deepCopyMap(in.Inventory.Labels),
		Annotations: deepCopyMap(in.Inventory.Annotations),
	}
	return out
}

func deepCopyMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// DeepCopyObject copies the receiver and creates a new runtime.Object.
func (in *Kptfile) DeepCopyObject() runtime.Object {
	if c := in.deepCopy(); c != nil {
		return c
	}
	return nil
}
