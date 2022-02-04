package rootsync

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func withSyncStatus(status v1beta1.SyncStatus) core.MetaMutator {
	return func(o client.Object) {
		rs := o.(*v1beta1.RootSync)
		rs.Status.SyncStatus = status
	}
}

func fakeSyncStatus() v1beta1.SyncStatus {
	return v1beta1.SyncStatus{
		Rendering: v1beta1.RenderingStatus{
			Errors: []v1beta1.ConfigSyncError{
				{Code: "1061", ErrorMessage: "rendering-error-message"},
			},
		},
		Source: v1beta1.GitSourceStatus{
			Errors: []v1beta1.ConfigSyncError{
				{Code: "1021", ErrorMessage: "1021-error-message"},
				{Code: "1022", ErrorMessage: "1022-error-message"},
			},
		},
		Sync: v1beta1.GitSyncStatus{
			Errors: []v1beta1.ConfigSyncError{
				{Code: "2009", ErrorMessage: "apiserver error"},
				{Code: "2009", ErrorMessage: "webhook error"},
			},
		},
	}
}

func TestErrors(t *testing.T) {
	testCases := []struct {
		name         string
		rs           *v1beta1.RootSync
		errorSources []v1beta1.ErrorSource
		want         []v1beta1.ConfigSyncError
	}{
		{
			name:         "errorSources is nil, rs is nil",
			rs:           nil,
			errorSources: nil,
			want:         nil,
		},
		{
			name:         "errorSources is nil, rs is not nil",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName),
			errorSources: nil,
			want:         nil,
		},
		{
			name:         "errorSources is not nil, rs is nil",
			rs:           nil,
			errorSources: []v1beta1.ErrorSource{v1beta1.RenderingError},
			want:         nil,
		},
		{
			name:         "errorSources = {}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{},
			want:         nil,
		},
		{
			name:         "errorSources = {RenderingError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.RenderingError},
			want: []v1beta1.ConfigSyncError{
				{Code: "1061", ErrorMessage: "rendering-error-message"},
			},
		},
		{
			name:         "errorSources = {SourceError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.SourceError},
			want: []v1beta1.ConfigSyncError{
				{Code: "1021", ErrorMessage: "1021-error-message"},
				{Code: "1022", ErrorMessage: "1022-error-message"},
			},
		},
		{
			name:         "errorSources = {SyncError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.SyncError},
			want: []v1beta1.ConfigSyncError{
				{Code: "2009", ErrorMessage: "apiserver error"},
				{Code: "2009", ErrorMessage: "webhook error"},
			},
		},
		{
			name:         "errorSources = {RenderingError, SourceError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.RenderingError, v1beta1.SourceError},
			want: []v1beta1.ConfigSyncError{
				{Code: "1061", ErrorMessage: "rendering-error-message"},
				{Code: "1021", ErrorMessage: "1021-error-message"},
				{Code: "1022", ErrorMessage: "1022-error-message"},
			},
		},
		{
			name:         "errorSources = {RenderingError, SyncError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.RenderingError, v1beta1.SyncError},
			want: []v1beta1.ConfigSyncError{
				{Code: "1061", ErrorMessage: "rendering-error-message"},
				{Code: "2009", ErrorMessage: "apiserver error"},
				{Code: "2009", ErrorMessage: "webhook error"},
			},
		},
		{
			name:         "errorSources = {SourceError, SyncError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.SourceError, v1beta1.SyncError},
			want: []v1beta1.ConfigSyncError{
				{Code: "1021", ErrorMessage: "1021-error-message"},
				{Code: "1022", ErrorMessage: "1022-error-message"},
				{Code: "2009", ErrorMessage: "apiserver error"},
				{Code: "2009", ErrorMessage: "webhook error"},
			},
		},
		{
			name:         "errorSources = {RenderingError, SourceError, SyncError}",
			rs:           fake.RootSyncObjectV1Beta1(configsync.RootSyncName, withSyncStatus(fakeSyncStatus())),
			errorSources: []v1beta1.ErrorSource{v1beta1.RenderingError, v1beta1.SourceError, v1beta1.SyncError},
			want: []v1beta1.ConfigSyncError{
				{Code: "1061", ErrorMessage: "rendering-error-message"},
				{Code: "1021", ErrorMessage: "1021-error-message"},
				{Code: "1022", ErrorMessage: "1022-error-message"},
				{Code: "2009", ErrorMessage: "apiserver error"},
				{Code: "2009", ErrorMessage: "webhook error"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := Errors(tc.rs, tc.errorSources)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Errors() got %v, want %v", got, tc.want)
			}
		})
	}
}
