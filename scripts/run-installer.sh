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

# The installer container.  The named registry should be publicly accessible.
readonly INSTALLER_CONTAINER="gcr.io/nomos-release/installer"

# If interactive is set to a nonempty string, the installer will use a
# menu-driven interactive installer.
INTERACTIVE=${INTERACTIVE:-""}

# The semantic verison number of the release to install.
VERSION=${VERSION:-"latest"}

# The default configuration to load.
CONFIG=${CONFIG:-configs/example.json}

# Change this to absolute path, so it can be mounted.
CONFIG="$(realpath $CONFIG)"

# The directory that is going to be used as output.
readonly output_dir=$(pwd)

config_dir="$(dirname ${CONFIG})"
config_container_dir="configs/"
if [[ "x${config_dir}" == "x" ]]; then
  # Avoid trying to mount a "" directory as a config directory: it's not going
  # to work.
  config_dir="${output_dir}"
  config_container_dir=""
fi

readonly config_filename="$(basename ${CONFIG})"

# Temporarily, pull the latest image to the local docker daemon cache.
# TODO(filmil): Remove this gimmick once we have a public cluster registry
# to pull from.
gcloud docker -- pull "${INSTALLER_CONTAINER}:${VERSION}"

echo "+++ Using output directory: ${output_dir}"
mkdir -p ${output_dir}/{kubeconfig,generated_certs,generated_configs,logs}
docker run -it \
  -u "$(id -u):$(id -g)" \
  -v "${HOME}":/home/user \
  -v "${output_dir}/generated_certs":/opt/installer/generated_certs \
  -v "${output_dir}/generated_configs":/opt/installer/generated_configs \
  -v "${output_dir}/kubeconfig":/opt/installer/kubeconfig \
  -v "${output_dir}/logs":/tmp \
  -v "${config_dir}":/opt/installer/configs \
  -e "INTERACTIVE=${INTERACTIVE}" \
  -e "VERSION=${VERSION}" \
  -e "CONFIG=${config_container_dir}${config_filename}" \
  -e "CONFIG_OUT=generated_configs/generated_config.yaml" \
  "${INSTALLER_CONTAINER}:${VERSION}" "$@"
echo "+++ Generated files are available in ${output_dir}"
