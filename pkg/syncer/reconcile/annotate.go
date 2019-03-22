package reconcile

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func kv(k, v string) keyVal {
	return keyVal{key: k, val: v}
}

type keyVal struct {
	key, val string
}

func annotateManaged(decl metav1.Object, importToken string) {
	annotate(decl,
		kv(v1.SyncTokenAnnotationKey, importToken),                 // Annotate the resource with the current version token.
		kv(v1.ResourceManagementKey, v1.ResourceManagementEnabled), // Annotate the resource as Nomos managed.
	)
}

func annotate(decl metav1.Object, annotations ...keyVal) {
	a := decl.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	for _, annotation := range annotations {
		a[annotation.key] = annotation.val
	}
	decl.SetAnnotations(a)
}
