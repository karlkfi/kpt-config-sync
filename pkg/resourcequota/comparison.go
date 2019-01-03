package resourcequota

import (
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// diffResourceLists returns the diff as a result of subtracting the sub list from the minuend list.
// i.e. if sub = 2 and minuend = 4, the diff is 2. If sub is 4 and minuend is 2, the diff is -2.
func diffResourceLists(sub, minuend corev1.ResourceList) corev1.ResourceList {
	diff := corev1.ResourceList{}

	for r, quantity := range minuend {
		if quantity.Cmp(sub[r]) != 0 {
			diffQ := quantity.DeepCopy()
			diffQ.Sub(sub[r])
			diff[r] = diffQ
		}
	}

	for r, quantity := range sub {
		if _, exists := minuend[r]; !exists {
			diffQ := quantity.DeepCopy()
			diffQ.Neg()
			diff[r] = diffQ
		}
	}
	return diff
}

// ResourceListsEqual return true if the resource lists are equal.
func ResourceListsEqual(lhs, rhs corev1.ResourceList) bool {
	return len(diffResourceLists(lhs, rhs)) == 0
}

// ResourceQuantityEqual provides a comparer option for resource.Quantity.
func ResourceQuantityEqual() cmp.Option {
	return cmp.Comparer(func(lhs, rhs resource.Quantity) bool {
		return lhs.Cmp(rhs) == 0
	})
}
