package controllers

import (
	"github.com/google/nomos/pkg/api/configsync"
	corev1 "k8s.io/api/core/v1"
)

const (
	// git-sync container specific environment variables.
	gitSyncName       = "GIT_SYNC_USERNAME"
	gitSyncPassword   = "GIT_SYNC_PASSWORD"
	gitSyncHTTPSProxy = "HTTPS_PROXY"
	gitSyncHTTPProxy  = "HTTP_PROXY"
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

// gitSyncHttpsProxyEnv returns environment variables for git-sync container for https_proxy env.
func gitSyncHTTPSProxyEnv(secretRef string, keys map[string]bool) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	if keys["https_proxy"] {
		httpsProxy := &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef,
				},
				Key: "https_proxy",
			},
		}
		envVars = append(envVars, corev1.EnvVar{
			Name:      gitSyncHTTPSProxy,
			ValueFrom: httpsProxy,
		})
	}
	if keys["http_proxy"] {
		httpProxy := &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef,
				},
				Key: "http_proxy",
			},
		}
		envVars = append(envVars, corev1.EnvVar{
			Name:      gitSyncHTTPProxy,
			ValueFrom: httpProxy,
		})
	}
	return envVars
}

func authTypeToken(secret string) bool {
	return configsync.GitSecretToken == secret
}
