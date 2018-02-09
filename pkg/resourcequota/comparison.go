package resourcequota

import (
	core_v1 "k8s.io/api/core/v1"
)

func resourceListEqual(left, right core_v1.ResourceList) bool {
	if len(left) != len(right) {
		return false
	}

	for resource, quantity := range left {
		if quantity.Cmp(right[resource]) != 0 {
			return false
		}
	}
	return true
}

// diffResourceLists returns the diff as a result of subtracting the sub list from the minuend list.
// i.e. if sub = 2 and minuend = 4, the diff is 2. If sub is 4 and minuend is 2, the diff is -2.
func diffResourceLists(sub, minuend core_v1.ResourceList) core_v1.ResourceList {
	diff := core_v1.ResourceList{}

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
