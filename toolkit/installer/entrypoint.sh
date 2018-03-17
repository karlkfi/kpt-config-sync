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

echo "+++ Installer version: ${INSTALLER_VERSION}"

# We need to fix up the kubeconfig paths because these may not match between
# the container and the host.
# /somepath/gcloud becomes /use/local/gcloud/google-cloud/sdk/bin/gcloud.
cat /home/user/.kube/config | \
  sed -e "s+cmd-path: [^ ]*gcloud+cmd-path: /usr/local/gcloud/google-cloud-sdk/bin/gcloud+g" \
  > /opt/installer/kubeconfig/config

if [ "x${INTERACTIVE}" != "x" ]; then
  echo "+++ Running configgen"
  ./configgen \
    --v=10 \
    --log_dir=/tmp \
    --version="${INSTALLER_VERSION}" \
    --config_in=${CONFIG} \
    --config_out=${CONFIG_OUT} \
    "$@"
else
  echo "+++ Running installer"
  ./installer \
    --v=10 \
    --log_dir=/tmp \
    --config="${CONFIG}" "$@"
fi

