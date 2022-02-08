// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"testing"
)

func TestGetNSFromSecret(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{
			name: "ns-reconciler-bookstore-token-1df",
			want: "bookstore",
		},
		{
			name: "ns-reconciler-bookstore-ssh-key",
			want: "bookstore",
		},
		{
			name: "ns-reconciler-bookstore-dfdd",
			want: "bookstore-dfdd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getNSFromSecret(tc.name)
			if tc.want != got {
				t.Errorf("getNSFromSecret(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestGetNSFromConfigMap(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{
			name: "ns-reconciler-bookstore-git-sync",
			want: "bookstore",
		},
		{
			name: "ns-reconciler-bookstore-reconciler",
			want: "bookstore",
		},
		{
			name: "ns-reconciler-bookstore-reconciler-git-sync",
			want: "bookstore",
		},
		{
			name: "ns-reconciler-bookstore-git-sync-reconciler",
			want: "bookstore-git-sync",
		},
		{
			name: "ns-reconciler-bookstore-dfdd",
			want: "bookstore-dfdd",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := getNSFromConfigMap(tc.name)
			if tc.want != got {
				t.Errorf("getNSFromConfigMap(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
