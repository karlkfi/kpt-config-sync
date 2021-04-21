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
