// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// Code generated by "stringer -type=PruneEventType"; DO NOT EDIT.

package event

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PruneEventResourceUpdate-0]
	_ = x[PruneEventCompleted-1]
	_ = x[PruneEventFailed-2]
}

const _PruneEventType_name = "PruneEventResourceUpdatePruneEventCompletedPruneEventFailed"

var _PruneEventType_index = [...]uint8{0, 24, 43, 59}

func (i PruneEventType) String() string {
	if i < 0 || i >= PruneEventType(len(_PruneEventType_index)-1) {
		return "PruneEventType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _PruneEventType_name[_PruneEventType_index[i]:_PruneEventType_index[i+1]]
}
