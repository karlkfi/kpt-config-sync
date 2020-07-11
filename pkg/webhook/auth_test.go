package webhook

import (
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"
)

func TestIsConfigSyncSA(t *testing.T) {
	testCases := []struct {
		name     string
		userInfo authenticationv1.UserInfo
		want     bool
	}{
		{
			"Config Sync service account",
			authenticationv1.UserInfo{
				Groups: []string{"foogroup", "system:serviceaccounts", "bargroup", "system:serviceaccounts:config-management-system", "bazgroup"},
			},
			true,
		},
		{
			"Gatekeeper service account",
			authenticationv1.UserInfo{
				Groups: []string{"system:serviceaccounts", "system:serviceaccounts:gatekeeper-system"},
			},
			false,
		},
		{
			"Invalid Config Sync service account",
			authenticationv1.UserInfo{
				Groups: []string{"foogroup", "system:serviceaccounts:config-management-system"},
			},
			false,
		},
		{
			"Invalid service account",
			authenticationv1.UserInfo{
				Groups: []string{"foogroup", "system:serviceaccounts"},
			},
			false,
		},
		{
			"Unauthenticated user",
			authenticationv1.UserInfo{
				Groups: []string{},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isConfigSyncSA(tc.userInfo); got != tc.want {
				t.Errorf("isConfigSyncSA got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestIsImporter(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		want     bool
	}{
		{
			"Config Sync importer service account",
			"system:serviceaccounts:config-management-system:importer",
			true,
		},
		{
			"Config Sync monitor service account",
			"system:serviceaccounts:config-management-system:monitor",
			false,
		},
		{
			"Random other service account named importer",
			"system:serviceaccounts:foo-namespace:importer",
			false,
		},
		{
			"Empty username",
			"",
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isImporter(tc.username); got != tc.want {
				t.Errorf("isImporter got %v; want %v", got, tc.want)
			}
		})
	}
}
