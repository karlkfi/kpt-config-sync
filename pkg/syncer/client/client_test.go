package client_test

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	syncertestfake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClient_Create(t *testing.T) {
	testCases := []struct {
		name     string
		declared core.Object
		client   client.Client
		wantErr  status.Error
	}{
		{
			name:     "Creates if does not exist",
			declared: fake.RoleObject(core.Name("admin"), core.Namespace("billing")),
			client:   syncertestfake.NewClient(t, runtime.NewScheme()),
			wantErr:  nil,
		},
		{
			name:     "Retriable if receives AlreadyExists",
			declared: fake.RoleObject(core.Name("admin"), core.Namespace("billing")),
			client: syncertestfake.NewClient(t, runtime.NewScheme(),
				fake.RoleObject(core.Name("admin"), core.Namespace("billing")),
			),
			wantErr: syncerclient.ConflictCreateAlreadyExists(errors.New("some error"),
				fake.RoleObject()),
		},
		{
			name:     "Generic APIServerError if other error",
			declared: fake.RoleObject(core.Name("admin"), core.Namespace("billing")),
			client:   syncertestfake.NewErrorClient(errors.New("some API server error")),
			wantErr:  status.APIServerError(errors.New("some error"), "could not create"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sc := syncerclient.New(tc.client, nil)

			err := sc.Create(context.Background(), tc.declared)
			if !errors.Is(tc.wantErr, err) {
				t.Fatalf("got err %v, want err %v", err, tc.wantErr)
			}
		})
	}
}
