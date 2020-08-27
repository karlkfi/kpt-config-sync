package controllers

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValidateSecretExist(t *testing.T) {
	testCases := []struct {
		name            string
		secretReference string
		secretNamespace string
		wantError       bool
		wantSecret      *corev1.Secret
	}{
		{
			name:            "Secret present",
			secretNamespace: "bookinfo",
			secretReference: "ssh-key",
			wantSecret:      secretObj(t, "ssh-key", secretAuth, core.Namespace("bookinfo")),
		},

		{
			name:            "Secret not present",
			secretNamespace: "bookinfo",
			secretReference: "ssh-key-root",
			wantError:       true,
		},
	}

	ctx := context.Background()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	fakeClient := syncerFake.NewClient(t, s, secretObj(t, "ssh-key", secretAuth, core.Namespace("bookinfo")))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secret, err := validateSecretExist(ctx, tc.secretReference, tc.secretNamespace, fakeClient)
			if tc.wantError && err == nil {
				t.Errorf("validateSecretExist() got error: %q, want error", err)
			} else if !tc.wantError && err != nil {
				t.Errorf("validateSecretExist() got error: %q, want error: nil", err)
			}
			if !tc.wantError {
				if diff := cmp.Diff(secret, tc.wantSecret); diff != "" {
					t.Errorf("mutateRepoSyncDeployment() got diff: %v\nwant: nil", diff)
				}
			}
		})
	}
}

func TestValidateSecretData(t *testing.T) {
	testCases := []struct {
		name      string
		auth      string
		secret    *corev1.Secret
		wantError bool
	}{
		{
			name:   "SSH auth data present",
			auth:   "ssh",
			secret: secretObj(t, "ssh-key", secretAuth, core.Namespace("bookinfo")),
		},
		{
			name:   "Cookiefile auth data present",
			auth:   "cookiefile",
			secret: secretObj(t, "ssh-key", "cookie_file", core.Namespace("bookinfo")),
		},
		{
			name: "None auth",
			auth: "none",
		},
		{
			name: "GCENode auth",
			auth: "gcenode",
		},
		{
			name:      "Usupported auth",
			auth:      "( ͡° ͜ʖ ͡°)",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSecretData(tc.auth, tc.secret)
			if tc.wantError && err == nil {
				t.Errorf("validateSecretData() got error: %q, want error", err)
			} else if !tc.wantError && err != nil {
				t.Errorf("validateSecretData() got error: %q, want error: nil", err)
			}
		})
	}
}
