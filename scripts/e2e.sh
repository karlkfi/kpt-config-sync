#!/usr/bin/env bash
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

E2E_CONTAINER="gcr.io/stolos-dev/e2e-tests"
VERSION=${VERSION:-"latest"}
CONFIG_DIR="examples"
OUTPUT_DIR=$(pwd)

docker pull "${E2E_CONTAINER}:${VERSION}"

echo "+++ Using output directory: ${OUTPUT_DIR}"
mkdir -p ${OUTPUT_DIR}/{kubeconfig,certs,gen_configs,logs}
docker run -it \
  -u "$(id -u):$(id -g)" \
  -v "${HOME}":/home/user \
  -v "${OUTPUT_DIR}/certs":/opt/installer/certs \
  -v "${OUTPUT_DIR}/gen_configs":/opt/installer/gen_configs \
  -v "${OUTPUT_DIR}/kubeconfig":/opt/installer/kubeconfig \
  -v "${OUTPUT_DIR}/logs":/tmp \
  -v "${CONFIG_DIR}":/opt/installer/configs \
  -e "VERSION=${VERSION}" \
  "${E2E_CONTAINER}:${VERSION}" "$@" \
	&& echo "+++ E2E tests completed." \
	|| (echo "### E2E tests failed. Logs are available in ${OUTPUT_DIR}/logs"; exit 1)
