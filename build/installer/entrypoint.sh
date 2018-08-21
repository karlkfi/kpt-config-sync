#!/bin/bash
#
# Copyright 2018 The Nomos Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License")
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script is the entry point for the installer container.  This script is
# intended to be ran only from within the installer container, which has a known
# path layout.

# Environment variables used in the entrypoint:
# - CONFIG: The name of the config file to use as default configuration,
# relative to the working directory inside the container.
# - HOME_ON_HOST: the string representing the absolute path of $HOME for the
# user that is running the installer.  In the container, we will soft-link this
# directory to /home/user.
# - INSTALLER_VERSION: the version of the installer to display in the installer
# backtitle.  Normally, this is not changed from the default value from the
# corresponding Dockerfile.   It does not affect what the installer runs, it's
# only for user information.

echo "### Running installer version: ${INSTALLER_VERSION}"

readonly kubeconfig_output="/opt/installer/kubeconfig/config"
# We need to fix up the kubeconfig paths because these may not match between
# the container and the host.
# /somepath/gcloud becomes /use/local/gcloud/google-cloud/sdk/bin/gcloud.
# Then, make it read-writable to the owner only.
cat /home/user/.kube/config | \
  sed -e "s+cmd-path: [^ ]*gcloud+cmd-path: /usr/local/gcloud/google-cloud-sdk/bin/gcloud+g" \
  > "${kubeconfig_output}"
chmod 600 ${kubeconfig_output}

# Only suggest the user where there is a gcloud utility available.
suggested_user=""
if command -v gcloud ; then
   suggested_user="$(gcloud config get-value account)"
else
   echo "gcloud is required."
   exit 1
fi

# Make sure the nomos-system k8s namespace exists before starting installation.
kubectl apply -f - << EOF
apiVersion: v1
kind: Namespace
metadata:
  name: nomos-system
  labels:
    nomos.dev/system: "true"
EOF

# Set logging levels to high for specific modules only.
readonly logging_options="--vmodule=main=5,configgen=3,kubectl=3,installer=3,exec=3"

# Fixes painting issues with running installer under GNU screen.  It doesn't
# seem to have ill effects on other terminals.
export TERM=screen

# Tells ncurses not to detect UTF-8 incapable terminals.  It misdiagnoses
# some cloud shells as unable to display UTF-8.
export NCURSES_NO_UTF8_ACS=1

echo "+++ Running installer"
INSTALLER_ARGS=(
  "${logging_options}"
  --config_out=gen_configs/generated.yaml
  --log_dir=/tmp
  --suggested_user="${suggested_user}"
  --version="${INSTALLER_VERSION}"
)
if [[ "${CONFIG}" != "" ]]; then
  CONTAINER_CONFIG_FILE="configs/config_in.yaml"
  INSTALLER_ARGS+=(
  	"--config=${CONTAINER_CONFIG_FILE}"
	"--config_in=${CONTAINER_CONFIG_FILE}"
  )
fi
./installer "${INSTALLER_ARGS[@]}" "$@"

