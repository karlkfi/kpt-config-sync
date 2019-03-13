package id

import (
	"fmt"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource identifies a Resource in a Nomos repository.
// Unique so long as no single file illegally declares two Resources of the same Name and Group/Version/Kind.
type Resource interface {
	// Sourced is the embedded interface providing path information to this Resource.
	nomospath.Sourced
	// Name returns the metadata.name of the Resource.
	Name() string
	// GroupVersionKind returns the K8S Group/Version/Kind of the Resource.
	GroupVersionKind() schema.GroupVersionKind
}

// PrintResource returns a human-readable output for the Resource.
func PrintResource(r Resource) string {
	return fmt.Sprintf("source: %[1]s\n"+
		"metadata.name:%[2]s\n"+
		"%[3]s",
		r.RelativeSlashPath(), name(r), printGroupVersionKind(r.GroupVersionKind()))
}

// name returns the empty string if r.Name is the empty string, otherwise prepends a space.
func name(r Resource) string {
	if r.Name() == "" {
		return ""
	}
	return " " + r.Name()
}

// ParseResource returns a Resource initialized from the given runtime.Object and a valid source
// path parsed from its annotations.
func ParseResource(object runtime.Object) Resource {
	mo := object.(metav1.Object)
	srcPath := mo.GetAnnotations()[v1.SourcePathAnnotationKey]
	return &resource{Object: object, srcPath: srcPath}
}

// resource is a base class that implements Resource
type resource struct {
	runtime.Object
	srcPath string
}

var _ Resource = &resource{}

// RelativeSlashPath implements nomospath.Sourced
func (r resource) RelativeSlashPath() string {
	return r.srcPath
}

// Name implements Resource
func (r resource) Name() string {
	return r.Object.(metav1.Object).GetName()
}

// GroupVersionKind implements Resource
func (r resource) GroupVersionKind() schema.GroupVersionKind {
	return r.GetObjectKind().GroupVersionKind()
}
