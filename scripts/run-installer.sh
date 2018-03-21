#!/bin/bash
#
# Copyright 2018 The Nomos Authors.
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

# The directory that is going to be used as output.
readonly output_dir=$(pwd)

# If interactive is set to a nonempty string, the installer will use a
# menu-driven interactive installer.
INTERACTIVE=${INTERACTIVE:-""}

# The semantic verison number of the release to install.
VERSION=${VERSION:-"latest"}

readonly gcloud_prg="$(which gcloud)"
if [[ -z "${gcloud_prg}" ]]; then
  echo "gcloud is required."
  exit 1
fi

CONFIG_DIR="examples"
CONFIG_BASENAME="quickstart.yaml"
if [[ -z "${CONFIG}" ]]; then
  CONTAINER_CONFIG_FILE="examples/quickstart.yaml"
else
  # $CONFIG specified, ensure that it's an existing file, and map it into the
  # container.
  CONFIG="$(realpath $CONFIG)"
  if ! [[ -f "${CONFIG}" ]]; then
    echo "file not found: ${CONFIG}"
    exit 1
  fi
  CONFIG_DIR="$(dirname $CONFIG)"
  CONFIG_BASENAME="$(basename $CONFIG)"
  CONTAINER_CONFIG_FILE="configs/${CONFIG_BASENAME}"
fi

# Only suggest the user where there is a gcloud utility available.
readonly suggested_user="$(gcloud config get-value account)"

# Temporarily, pull the latest image to the local docker daemon cache.
# TODO(filmil): Remove this gimmick once we have a public cluster registry
# to pull from.
gcloud docker -- pull "${INSTALLER_CONTAINER}:${VERSION}"

# TODO(filmil): Move the environment variables to flags.
echo "+++ Using output directory: ${output_dir}"
mkdir -p ${output_dir}/{kubeconfig,certs,gen_configs,logs}
docker run -it \
  -u "$(id -u):$(id -g)" \
  -v "${HOME}":/home/user \
  -v "${output_dir}/certs":/opt/installer/certs \
  -v "${output_dir}/gen_configs":/opt/installer/gen_configs \
  -v "${output_dir}/kubeconfig":/opt/installer/kubeconfig \
  -v "${output_dir}/logs":/tmp \
  -v "${CONFIG_DIR}":/opt/installer/configs \
  -e "INTERACTIVE=${INTERACTIVE}" \
  -e "VERSION=${VERSION}" \
  -e "CONFIG=${CONTAINER_CONFIG_FILE}" \
  -e "CONFIG_OUT=gen_configs/generated.yaml" \
  -e "SUGGESTED_USER=${suggested_user}" \
  "${INSTALLER_CONTAINER}:${VERSION}" "$@"
echo "+++ Generated files are available in ${output_dir}"

