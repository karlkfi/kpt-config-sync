package resourcequota

import (
	corev1 "k8s.io/api/core/v1"
)

// diffResourceLists returns the diff as a result of subtracting the sub list from the minuend list.
// i.e. if sub = 2 and minuend = 4, the diff is 2. If sub is 4 and minuend is 2, the diff is -2.
func diffResourceLists(sub, minuend corev1.ResourceList) corev1.ResourceList {
	diff := corev1.ResourceList{}

	for resource, quantity := range minuend {
		if quantity.Cmp(sub[resource]) != 0 {
			diffQ := quantity.DeepCopy()
			diffQ.Sub(sub[resource])
			diff[resource] = diffQ
		}
	}

	for resource, quantity := range sub {
		if _, exists := minuend[resource]; !exists {
			diffQ := quantity.DeepCopy()
			diffQ.Neg()
			diff[resource] = diffQ
		}
	}
	return diff
}

// ResourceListsEqual return true if the resource lists are equal.
func ResourceListsEqual(lhs, rhs corev1.ResourceList) bool {
	return len(diffResourceLists(lhs, rhs)) == 0
}
