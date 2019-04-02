package reconcile

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func kv(k, v string) keyVal {
	return keyVal{key: k, val: v}
}

type keyVal struct {
	key, val string
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
