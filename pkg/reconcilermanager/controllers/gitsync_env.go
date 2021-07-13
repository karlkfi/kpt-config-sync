package controllers

import (
	"github.com/google/nomos/pkg/api/configsync"
	corev1 "k8s.io/api/core/v1"
)

const (
	// git-sync container specific environment variables.
	gitSyncName     = "GIT_SYNC_USERNAME"
	gitSyncPassword = "GIT_SYNC_PASSWORD"
)

// gitSyncTokenAuthEnv returns environment variables for git-sync container for 'token' Auth.
func gitSyncTokenAuthEnv(secretRef string) []corev1.EnvVar {
	gitSyncUsername := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretRef,
			},
			Key: "username",
		},
	}

	gitSyncPswd := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretRef,
			},
			Key: "token",
		},
	}

	return []corev1.EnvVar{
		{
			Name:      gitSyncName,
			ValueFrom: gitSyncUsername,
		},
		{
			Name:      gitSyncPassword,
			ValueFrom: gitSyncPswd,
		},
	}
}

func authTypeToken(secret string) bool {
	return configsync.GitSecretToken == secret
}
