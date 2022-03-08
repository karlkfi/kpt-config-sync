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
	corev1 "k8s.io/api/core/v1"
	"kpt.dev/configsync/pkg/api/configsync"
)

const (
	// git-sync container specific environment variables.
	gitSyncName       = "GIT_SYNC_USERNAME"
	gitSyncPassword   = "GIT_SYNC_PASSWORD"
	gitSyncHTTPSProxy = "HTTPS_PROXY"
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
	return envVars
}

func authTypeToken(secret string) bool {
	return configsync.GitSecretToken == secret
}
