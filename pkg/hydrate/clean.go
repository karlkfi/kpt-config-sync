package hydrate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/validate/raw/validate"
)

// Clean removes invalid fields from objects before writing them to a file.
func Clean(objects []ast.FileObject) {
	for _, o := range objects {
		clean(o)
	}
}

func clean(o ast.FileObject) {
	annotations := o.GetAnnotations()

	// Restore or remove the hnc.x-k8s.io/managed-by annotation from namespace objects.
	if o.GetObjectKind().GroupVersionKind() == kinds.Namespace() {
		for k := range annotations {
			if k == hnc.AnnotationKeyV1A2 {
				if value, ok := annotations[hnc.OriginalHNCManagedByValue]; ok {
					annotations[hnc.AnnotationKeyV1A2] = value
				} else {
					delete(annotations, k)
				}
			}
		}
	}

	// Remove all the annotations starting with configsync.gke.io or configmanagement.gke.io
	// except for the configmanagement.gke.io/managed annotation.
	for k := range annotations {
		if validate.HasConfigSyncPrefix(k) && k != v1.ResourceManagementKey {
			delete(annotations, k)
		}
	}

	if len(annotations) == 0 {
		// Set annotations to nil so that the `annotations` field can be removed from the object metadata.
		annotations = nil
	}
	o.SetAnnotations(annotations)

	labels := o.GetLabels()

	// Remove the <ns>.tree.hnc.x-k8s.io/depth label(s) from namespace objects.
	if o.GetObjectKind().GroupVersionKind() == kinds.Namespace() {
		for k := range labels {
			if validate.HasDepthSuffix(k) {
				delete(labels, k)
			}
		}
	}

	// Remove all the labels starting with configsync.gke.io or configmanagement.gke.io.
	for k := range labels {
		if validate.HasConfigSyncPrefix(k) {
			delete(labels, k)
		}
	}

	if len(labels) == 0 {
		// Set labels to nil so that the `labels` field can be removed from the object metadata.
		labels = nil
	}
	o.SetLabels(labels)
}
