#!/bin/bash
#
# Copyright 2018 The Stolos Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
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

# For debugging.  Remove before release.
set -x

# If interactive is set to a nonempty string, the installer will use a
# menu-driven interactive installer.
INTERACTIVE=${INTERACTIVE:-""}

# The semantic verison number of the release to install.
VERSION=${VERSION:-"latest"}

# The default configuration to load.
CONFIG=${CONFIG:-configs/example.json}

readonly config_dir="$(dirname ${CONFIG})"
readonly config_filename="$(basename ${CONFIG})"

# The directory that is going to be used as output.
readonly tempdir=$(mktemp -d)

echo "+++ Using directory: ${tempdir}"
mkdir -p ${tempdir}/{kubeconfig,generated_certs,generated_configs}
docker run -it \
  -u "$(id -u):$(id -g)" \
  -v "${HOME}":/home/user \
  -v "${tempdir}/generated_certs":/opt/installer/generated_certs \
  -v "${tempdir}/kubeconfig":/opt/installer/kubeconfig \
  -v "${config_dir}":/opt/installer/configs \
  -e "INTERACTIVE=${INTERACTIVE}" \
  -e "VERSION=${VERSION}" \
  -e "CONFIG=configs/${config_filename}" \
  -e "CONFIG_OUT=generated_configs/generated_config.json" \
  gcr.io/stolos-dev/installer:${VERSION}
echo "+++ Generated certs are in ${tempdir}"
