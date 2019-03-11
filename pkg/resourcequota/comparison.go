package resourcequota

import (
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ResourceQuantityEqual allows the resource.Quantity to be checked for equality and allow the fields of
// ast.FileObject to be printed nicely when passed to cmp.Diff.
func ResourceQuantityEqual() cmp.Option {
	return cmp.Comparer(func(x, y resource.Quantity) bool {
		return x.Cmp(y) == 0
	})
}
