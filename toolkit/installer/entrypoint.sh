#!/bin/bash
#
# Copyright 2018 The Stolos Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License"};
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
# - INTERACTIVE: If set to a nonempty string, runs the menu-driven installer.
#   Otherwise, the batch installer is ran.
# - CONFIG: The name of the config file to use as default configuration.
# - CONFIG_OUT: The name of the config file to use as output.

echo "##################################################"
echo "+++ Installer script version: ${INSTALLER_VERSION}"
echo "##################################################"

echo "### Examining environment:"
env

echo "### Examining args"
echo "### $@"

readonly kubeconfig_output="/opt/installer/kubeconfig/config"
# We need to fix up the kubeconfig paths because these may not match between
# the container and the host.
# /somepath/gcloud becomes /use/local/gcloud/google-cloud/sdk/bin/gcloud.
# Then, make it read-writable to the owner only.
cat /home/user/.kube/config | \
  sed -e "s+cmd-path: [^ ]*gcloud+cmd-path: /usr/local/gcloud/google-cloud-sdk/bin/gcloud+g" \
  > "${kubeconfig_output}"
chmod 600 ${kubeconfig_output}

# Set logging levels to high for specific modules only.
readonly logging_options="--vmodule=main=10,configgen=10,kubectl=10,installer=10,exec=10,menu=10"

if [ "x${INTERACTIVE}" != "x" ]; then
  echo "+++ Running configgen"
  ./configgen \
    ${logging_options} \
    --log_dir=/tmp \
    --suggested_user="${SUGGESTED_USER}" \
    --version="${INSTALLER_VERSION}" \
    --config_in=${CONFIG} \
    --config_out=${CONFIG_OUT} \
    "$@"
else
  echo "+++ Running installer"
  ./installer \
    ${logging_options} \
    --log_dir=/tmp \
    --config="${CONFIG}" "$@"
fi

