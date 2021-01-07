// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// Code generated by "stringer -type=smpDirective -linecomment"; DO NOT EDIT.

package merge2

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[smpUnknown-0]
	_ = x[smpReplace-1]
	_ = x[smpDelete-2]
	_ = x[smpMerge-3]
}

const _smpDirective_name = "unknownreplacedeletemerge"

var _smpDirective_index = [...]uint8{0, 7, 14, 20, 25}

func (i smpDirective) String() string {
	if i < 0 || i >= smpDirective(len(_smpDirective_index)-1) {
		return "smpDirective(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _smpDirective_name[_smpDirective_index[i]:_smpDirective_index[i+1]]
}
